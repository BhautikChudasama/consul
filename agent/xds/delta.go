// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package xds

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/armon/go-metrics"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/go-hclog"
	goversion "github.com/hashicorp/go-version"

	"github.com/hashicorp/consul/agent/envoyextensions"
	"github.com/hashicorp/consul/agent/proxycfg"
	"github.com/hashicorp/consul/agent/xds/configfetcher"
	"github.com/hashicorp/consul/agent/xds/extensionruntime"
	"github.com/hashicorp/consul/agent/xdsv2"
	"github.com/hashicorp/consul/envoyextensions/extensioncommon"
	"github.com/hashicorp/consul/envoyextensions/xdscommon"
	proxysnapshot "github.com/hashicorp/consul/internal/mesh/proxy-snapshot"
	proxytracker "github.com/hashicorp/consul/internal/mesh/proxy-tracker"
	"github.com/hashicorp/consul/logging"
	pbmesh "github.com/hashicorp/consul/proto-public/pbmesh/v2beta1"
	"github.com/hashicorp/consul/proto-public/pbresource"
	"github.com/hashicorp/consul/version"
)

var errOverwhelmed = status.Error(codes.ResourceExhausted, "this server has too many xDS streams open, please try another")

// xdsProtocolLegacyChildResend enables the legacy behavior for the `ensureChildResend` function.
// This environment variable exists as an escape hatch so that users can disable the behavior, if needed.
// Ideally, this is a flag we can remove in 1.19+
var xdsProtocolLegacyChildResend = (os.Getenv("XDS_PROTOCOL_LEGACY_CHILD_RESEND") != "")

type deltaRecvResponse int

const (
	deltaRecvResponseNack deltaRecvResponse = iota
	deltaRecvResponseAck
	deltaRecvNewSubscription
	deltaRecvUnknownType
)

// ADSDeltaStream is a shorter way of referring to this thing...
type ADSDeltaStream = envoy_discovery_v3.AggregatedDiscoveryService_DeltaAggregatedResourcesServer

// DeltaAggregatedResources implements envoy_discovery_v3.AggregatedDiscoveryServiceServer
func (s *Server) DeltaAggregatedResources(stream ADSDeltaStream) error {
	defer s.activeStreams.Increment(stream.Context())()

	// a channel for receiving incoming requests
	reqCh := make(chan *envoy_discovery_v3.DeltaDiscoveryRequest)
	reqStop := int32(0)
	go func() {
		for {
			req, err := stream.Recv()
			if atomic.LoadInt32(&reqStop) != 0 {
				return
			}
			if err != nil {
				s.Logger.Error("Error receiving new DeltaDiscoveryRequest; closing request channel", "error", err)
				close(reqCh)
				return
			}

			select {
			case <-stream.Context().Done():
			case reqCh <- req:
			}
		}
	}()

	err := s.processDelta(stream, reqCh)
	if err != nil {
		s.Logger.Error("Error handling ADS delta stream", "xdsVersion", "v3", "error", err)
	}

	// prevents writing to a closed channel if send failed on blocked recv
	atomic.StoreInt32(&reqStop, 1)

	return err
}

// getEnvoyConfiguration is a utility function that instantiates the proper
// Envoy resource generator based on whether it was passed a ConfigSource or
// ProxyState implementation of the ProxySnapshot interface and returns the
// generated Envoy configuration.
func getEnvoyConfiguration(proxySnapshot proxysnapshot.ProxySnapshot, logger hclog.Logger, cfgFetcher configfetcher.ConfigFetcher) (map[string][]proto.Message, error) {
	switch proxySnapshot.(type) {
	case *proxycfg.ConfigSnapshot:
		logger.Trace("ProxySnapshot update channel received a ProxySnapshot of type ConfigSnapshot")
		generator := NewResourceGenerator(
			logger,
			cfgFetcher,
			true,
		)

		c := proxySnapshot.(*proxycfg.ConfigSnapshot)
		return generator.AllResourcesFromSnapshot(c)
	case *proxytracker.ProxyState:
		logger.Trace("ProxySnapshot update channel received a ProxySnapshot of type ProxyState")
		generator := xdsv2.NewResourceGenerator(
			logger,
		)
		c := proxySnapshot.(*proxytracker.ProxyState)
		resources, err := generator.AllResourcesFromIR(c)
		if err != nil {
			logger.Error("error generating resources from proxy state template", "err", err)
			return nil, err
		}
		return resources, nil
	default:
		return nil, errors.New("proxysnapshot must be of type ProxyState or ConfigSnapshot")
	}
}

func (s *Server) processDelta(stream ADSDeltaStream, reqCh <-chan *envoy_discovery_v3.DeltaDiscoveryRequest) error {
	// Handle invalid ACL tokens up-front.
	if _, err := s.authenticate(stream.Context()); err != nil {
		return err
	}

	// Loop state
	session := newDeltaSession(
		s.Logger.Named(logging.XDS).With("xdsVersion", "v3"),
		stream,
	)
	defer session.Stop()

	var authTimer <-chan time.Time
	extendAuthTimer := func() {
		authTimer = time.After(s.AuthCheckFrequency)
	}

	for {
		select {
		case <-session.drainCh:
			session.logger.Debug("draining stream to rebalance load")
			metrics.IncrCounter([]string{"xds", "server", "streamDrained"}, 1)
			return errOverwhelmed

		case <-authTimer:
			// It's been too long since a Discovery{Request,Response} so recheck ACLs.
			if err := session.CheckStreamACLs(s); err != nil {
				return err
			}
			extendAuthTimer()
			continue

		case req, ok := <-reqCh:
			if !ok {
				// reqCh is closed when stream.Recv errors which is how we detect client
				// going away. AFAICT the stream.Context() is only canceled once the
				// RPC method returns which it can't until we return from this one so
				// there's no point in blocking on that.
				return nil
			}

			regen, err := session.AcceptDiscoveryRequest(req)
			if err != nil {
				return err
			} else if !regen {
				continue
			}

			if !session.ready {
				if err := session.InitializeWatch(s); err != nil {
					return err
				}
				// Now wait for the config so we can check ACL
				continue
			}

		case cs, ok := <-session.stateCh:
			if !ok {
				// stateCh is closed either when *we* cancel the watch (on-exit via defer)
				// or by the proxycfg.Manager when an irrecoverable error is encountered
				// such as the ACL token getting deleted.
				//
				// We know for sure that this is the latter case, because in the former we
				// would've already exited this loop.
				return status.Error(codes.Aborted, "xDS stream terminated due to an irrecoverable error, please try again")
			}

			if err := session.InstallSnapshot(s, cs); err != nil {
				return err
			}
		}

		// At this point the session is ready.

		// Check ACLs on every Discovery{Request,Response}.
		if err := session.CheckStreamACLs(s); err != nil {
			return err
		}
		extendAuthTimer() // For the first time through, this is when the timer is first started.

		if err := session.UpdateEnvoyIfNecessary(); err != nil {
			return err
		}
	}
}

// newResourceIDFromEnvoyNode is a utility function that allows creating a
// Resource ID from an Envoy proxy node so that existing delta calls can easily
// use ProxyWatcher interface arguments for Watch().
func newResourceIDFromEnvoyNode(node *envoy_config_core_v3.Node) *pbresource.ID {
	entMeta := parseEnterpriseMeta(node)

	return &pbresource.ID{
		Name: node.Id,
		Tenancy: &pbresource.Tenancy{
			Namespace: entMeta.NamespaceOrDefault(),
			Partition: entMeta.PartitionOrDefault(),
		},
		Type: pbmesh.ProxyStateTemplateType,
	}
}

func (s *Server) applyEnvoyExtensions(resources *xdscommon.IndexedResources, proxySnapshot proxysnapshot.ProxySnapshot, node *envoy_config_core_v3.Node) (*xdscommon.IndexedResources, error) {
	// TODO(proxystate)
	// This is a workaround for now as envoy extensions are not yet supported with ProxyState.
	// For now, we cast to proxycfg.ConfigSnapshot and no-op if it's the pbmesh.ProxyState type.
	var snapshot *proxycfg.ConfigSnapshot
	switch proxySnapshot.(type) {
	//TODO(proxystate): implement envoy extensions for ProxyState
	case *proxytracker.ProxyState:
		return resources, nil
	case *proxycfg.ConfigSnapshot:
		snapshot = proxySnapshot.(*proxycfg.ConfigSnapshot)
	default:
		return nil, status.Errorf(codes.InvalidArgument,
			"unsupported config snapshot type to apply envoy extensions to %T", proxySnapshot)
	}
	var err error
	envoyVersion := xdscommon.DetermineEnvoyVersionFromNode(node)
	consulVersion, err := goversion.NewVersion(version.Version)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse Consul version")
	}

	serviceConfigs := extensionruntime.GetRuntimeConfigurations(snapshot)
	for _, cfgs := range serviceConfigs {
		for _, cfg := range cfgs {
			resources, err = validateAndApplyEnvoyExtension(s.Logger, snapshot, resources, cfg, envoyVersion, consulVersion)

			if err != nil {
				return nil, err
			}
		}
	}

	return resources, nil
}

func validateAndApplyEnvoyExtension(logger hclog.Logger, cfgSnap *proxycfg.ConfigSnapshot, resources *xdscommon.IndexedResources, runtimeConfig extensioncommon.RuntimeConfig, envoyVersion, consulVersion *goversion.Version) (*xdscommon.IndexedResources, error) {
	logFn := logger.Warn
	if runtimeConfig.EnvoyExtension.Required {
		logFn = logger.Error
	}

	svc := runtimeConfig.ServiceName

	errorParams := []interface{}{
		"extension", runtimeConfig.EnvoyExtension.Name,
		"service", svc.Name,
		"namespace", svc.Namespace,
		"partition", svc.Partition,
	}

	getMetricLabels := func(err error) []metrics.Label {
		return []metrics.Label{
			{Name: "extension", Value: runtimeConfig.EnvoyExtension.Name},
			{Name: "version", Value: "builtin/" + version.Version},
			{Name: "service", Value: cfgSnap.Service},
			{Name: "partition", Value: cfgSnap.ProxyID.PartitionOrDefault()},
			{Name: "namespace", Value: cfgSnap.ProxyID.NamespaceOrDefault()},
			{Name: "error", Value: strconv.FormatBool(err != nil)},
		}
	}

	ext := runtimeConfig.EnvoyExtension

	if v := ext.EnvoyVersion; v != "" {
		c, err := goversion.NewConstraint(v)
		if err != nil {
			logFn("failed to parse Envoy extension version constraint", errorParams...)

			if ext.Required {
				return nil, status.Errorf(codes.InvalidArgument, "failed to parse Envoy version constraint for extension %q for service %q", ext.Name, svc.Name)
			}
			return resources, nil
		}

		if !c.Check(envoyVersion) {
			logger.Info("skipping envoy extension due to Envoy version constraint violation", errorParams...)
			return resources, nil
		}
	}

	if v := ext.ConsulVersion; v != "" {
		c, err := goversion.NewConstraint(v)
		if err != nil {
			logFn("failed to parse Consul extension version constraint", errorParams...)

			if ext.Required {
				return nil, status.Errorf(codes.InvalidArgument, "failed to parse Consul version constraint for extension %q for service %q", ext.Name, svc.Name)
			}
			return resources, nil
		}

		if !c.Check(consulVersion) {
			logger.Info("skipping envoy extension due to Consul version constraint violation", errorParams...)
			return resources, nil
		}
	}

	now := time.Now()
	extender, err := envoyextensions.ConstructExtension(ext)
	metrics.MeasureSinceWithLabels([]string{"envoy_extension", "validate_arguments"}, now, getMetricLabels(err))
	if err != nil {
		errorParams = append(errorParams, "error", err)
		logFn("failed to construct extension", errorParams...)

		if ext.Required {
			return nil, status.Errorf(codes.InvalidArgument, "failed to construct extension %q for service %q", ext.Name, svc.Name)
		}

		return resources, nil
	}

	now = time.Now()
	err = extender.Validate(&runtimeConfig)
	metrics.MeasureSinceWithLabels([]string{"envoy_extension", "validate"}, now, getMetricLabels(err))
	if err != nil {
		errorParams = append(errorParams, "error", err)
		logFn("failed to validate extension arguments", errorParams...)

		if ext.Required {
			return nil, status.Errorf(codes.InvalidArgument, "failed to validate arguments for extension %q for service %q", ext.Name, svc.Name)
		}

		return resources, nil
	}

	now = time.Now()
	resources, err = applyEnvoyExtension(extender, resources, &runtimeConfig)
	metrics.MeasureSinceWithLabels([]string{"envoy_extension", "extend"}, now, getMetricLabels(err))
	if err != nil {
		errorParams = append(errorParams, "error", err)
		logFn("failed to apply envoy extension", errorParams...)

		if ext.Required {
			return nil, status.Errorf(codes.InvalidArgument, "failed to patch xDS resources in the %q extension: %v", ext.Name, err)
		}
	}

	return resources, nil
}

// applyEnvoyExtension safely checks whether an extension can be applied, and if so attempts to apply it.
//
// applyEnvoyExtension makes a copy of the provided IndexedResources, then applies the given extension to them.
// The copy ensures against partial application if a non-required extension modifies a resource then fails at a later
// stage; this is necessary because IndexedResources and its proto messages are all passed by reference, and
// non-required extensions do not lead to a terminal failure in xDS updates.
//
// If the application is successful, the modified copy is returned. If not, the original and an error is returned.
// Returning resources in either case allows for applying extensions in a loop and reporting on non-required extension
// failures simultaneously.
func applyEnvoyExtension(extender extensioncommon.EnvoyExtender, resources *xdscommon.IndexedResources, runtimeConfig *extensioncommon.RuntimeConfig) (r *xdscommon.IndexedResources, e error) {
	// Don't panic due to an extension misbehaving.
	defer func() {
		if err := recover(); err != nil {
			r = resources
			e = fmt.Errorf("attempt to apply Envoy extension %q caused an unexpected panic: %v",
				runtimeConfig.EnvoyExtension.Name, err)
		}
	}()

	// First check whether the extension is eligible for application in the current environment.
	// Do this before copying indexed resources for the sake of efficiency.
	if !extender.CanApply(runtimeConfig) {
		return resources, nil
	}

	newResources, err := extender.Extend(xdscommon.Clone(resources), runtimeConfig)
	if err != nil {
		return resources, err
	}

	return newResources, nil
}

// https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#eventual-consistency-considerations
var xDSUpdateOrder = []xDSUpdateOperation{
	// 1. SDS updates (if any) can be pushed here with no harm.
	{TypeUrl: xdscommon.SecretType, Upsert: true},
	// 2. CDS updates (if any) must always be pushed before the following types.
	{TypeUrl: xdscommon.ClusterType, Upsert: true},
	// 3. EDS updates (if any) must arrive after CDS updates for the respective clusters.
	{TypeUrl: xdscommon.EndpointType, Upsert: true},
	// 4. LDS updates must arrive after corresponding CDS/EDS updates.
	{TypeUrl: xdscommon.ListenerType, Upsert: true, Remove: true},
	// 5. RDS updates related to the newly added listeners must arrive after CDS/EDS/LDS updates.
	{TypeUrl: xdscommon.RouteType, Upsert: true, Remove: true},
	// 6. (NOT IMPLEMENTED YET IN CONSUL) VHDS updates (if any) related to the newly added RouteConfigurations must arrive after RDS updates.
	// {},
	// 7. Stale CDS clusters, related EDS endpoints (ones no longer being referenced) and SDS secrets can then be removed.
	{TypeUrl: xdscommon.ClusterType, Remove: true},
	{TypeUrl: xdscommon.EndpointType, Remove: true},
	{TypeUrl: xdscommon.SecretType, Remove: true},
	// xDS updates can be pushed independently if no new
	// clusters/routes/listeners are added or if it’s acceptable to
	// temporarily drop traffic during updates. Note that in case of
	// LDS updates, the listeners will be warmed before they receive
	// traffic, i.e. the dependent routes are fetched through RDS if
	// configured. Clusters are warmed when adding/removing/updating
	// clusters. On the other hand, routes are not warmed, i.e., the
	// management plane must ensure that clusters referenced by a route
	// are in place, before pushing the updates for a route.
}

type xDSUpdateOperation struct {
	TypeUrl string
	Upsert  bool
	Remove  bool
}

func (op *xDSUpdateOperation) errorLogNameReplyPrefix() string {
	switch {
	case op.Upsert && op.Remove:
		return "upsert/remove "
	case op.Upsert:
		return "upsert "
	case op.Remove:
		return "remove "
	default:
		return ""
	}
}

type xDSDeltaChild struct {
	// childType is a type that in Envoy is actually stored within this type.
	// Upserts of THIS type should potentially trigger dependent named
	// resources within the child to be re-configured.
	childType *xDSDeltaType

	// childrenNames is map of parent resource names to a list of associated child resource
	// names.
	childrenNames map[string][]string
}

type xDSDeltaType struct {
	logger       hclog.Logger
	stream       ADSDeltaStream
	typeURL      string
	allowEmptyFn func() bool

	// deltaChild contains data for an xDS child type if there is one.
	// For example, endpoints are a child type of clusters.
	deltaChild *xDSDeltaChild

	// registered indicates if this type has been requested at least once by
	// the proxy
	registered bool

	// wildcard indicates that this type was requested with no preference for
	// specific resource names. subscribe/unsubscribe are ignored.
	wildcard bool

	// sentToEnvoyOnce is true after we've sent one response to envoy.
	sentToEnvoyOnce bool

	// subscriptions is the set of currently subscribed envoy resources.
	// If wildcard == true, this will be empty.
	subscriptions map[string]struct{}

	// resourceVersions is the current view of CONFIRMED/ACKed updates to
	// envoy's view of the loaded resources.
	//
	// name => version
	resourceVersions map[string]string

	// pendingUpdates is a set of un-ACKed updates to the 'resourceVersions'
	// map. Once we get an ACK from envoy we'll update the resourceVersions map
	// and strike the entry from this map.
	//
	// nonce -> name -> {version}
	pendingUpdates map[string]map[string]PendingUpdate
}

func (t *xDSDeltaType) subscribed(name string) bool {
	if t.wildcard {
		return true
	}
	_, subscribed := t.subscriptions[name]
	return subscribed
}

type PendingUpdate struct {
	Remove  bool
	Version string
}

func newDeltaType(
	logger hclog.Logger,
	stream ADSDeltaStream,
	typeUrl string,
	allowEmptyFn func() bool,
) *xDSDeltaType {
	return &xDSDeltaType{
		logger:           logger,
		stream:           stream,
		typeURL:          typeUrl,
		allowEmptyFn:     allowEmptyFn,
		subscriptions:    make(map[string]struct{}),
		resourceVersions: make(map[string]string),
		pendingUpdates:   make(map[string]map[string]PendingUpdate),
	}
}

// Recv handles new discovery requests from envoy.
//
// Returns true the first time a type receives a request.
func (t *xDSDeltaType) Recv(req *envoy_discovery_v3.DeltaDiscoveryRequest, sf xdscommon.SupportedProxyFeatures) deltaRecvResponse {
	if t == nil {
		return deltaRecvUnknownType // not something we care about
	}

	registeredThisTime := false
	if !t.registered {
		// We are in the wildcard mode if the first request of a particular
		// type has empty subscription list
		t.wildcard = len(req.ResourceNamesSubscribe) == 0
		t.registered = true
		registeredThisTime = true
	}

	/*
		DeltaDiscoveryRequest can be sent in the following situations:

		Initial message in a xDS bidirectional gRPC stream.

		As an ACK or NACK response to a previous DeltaDiscoveryResponse. In
		this case the response_nonce is set to the nonce value in the Response.
		ACK or NACK is determined by the absence or presence of error_detail.

		Spontaneous DeltaDiscoveryRequests from the client. This can be done to
		dynamically add or remove elements from the tracked resource_names set.
		In this case response_nonce must be omitted.

	*/

	/*
		DeltaDiscoveryRequest plays two independent roles. Any
		DeltaDiscoveryRequest can be either or both of:
	*/

	if req.ResponseNonce != "" {
		/*
			[2] (N)ACKing an earlier resource update from the server (using
			response_nonce, with presence of error_detail making it a NACK).
		*/
		if req.ErrorDetail == nil {
			t.logger.Trace("got ok response from envoy proxy", "nonce", req.ResponseNonce)
			t.ack(req.ResponseNonce)
		} else {
			t.logger.Error("got error response from envoy proxy", "nonce", req.ResponseNonce,
				"error", status.ErrorProto(req.ErrorDetail))
			t.nack(req.ResponseNonce)
			return deltaRecvResponseNack
		}
	}

	if registeredThisTime && len(req.InitialResourceVersions) > 0 {
		/*
			Additionally, the first message (for a given type_url) of a
			reconnected gRPC stream has a third role:

			[3] informing the server of the resources (and their versions) that
			the client already possesses, using the initial_resource_versions
			field.
		*/
		t.logger.Trace("setting initial resource versions for stream",
			"resources", req.InitialResourceVersions)
		t.resourceVersions = req.InitialResourceVersions
		if !t.wildcard {
			for k := range req.InitialResourceVersions {
				t.subscriptions[k] = struct{}{}
			}
		}
	}

	if !t.wildcard {
		/*
			[1] informing the server of what resources the client has
			gained/lost interest in (using resource_names_subscribe and
			resource_names_unsubscribe), or
		*/
		for _, name := range req.ResourceNamesSubscribe {
			// A resource_names_subscribe field may contain resource names that
			// the server believes the client is already subscribed to, and
			// furthermore has the most recent versions of. However, the server
			// must still provide those resources in the response; due to
			// implementation details hidden from the server, the client may
			// have “forgotten” those resources despite apparently remaining
			// subscribed.
			//
			// NOTE: the server must respond with all resources listed in
			// resource_names_subscribe, even if it believes the client has the
			// most recent version of them. The reason: the client may have
			// dropped them, but then regained interest before it had a chance
			// to send the unsubscribe message.
			//
			// We handle that here by ALWAYS wiping the version so the diff
			// decides to send the value.
			_, alreadySubscribed := t.subscriptions[name]
			t.subscriptions[name] = struct{}{}

			// Reset the tracked version so we force a reply.
			if _, alreadyTracked := t.resourceVersions[name]; alreadyTracked {
				t.resourceVersions[name] = ""
			}

			// Certain xDS types are children of other types, meaning that if Envoy subscribes to a parent.
			// We MUST assume that if Envoy ever had data for the children of this parent, then the child's
			// data is gone.
			if t.deltaChild != nil && t.deltaChild.childType.registered {
				for _, childName := range t.deltaChild.childrenNames[name] {
					t.ensureChildResend(name, childName)
				}
			}

			if alreadySubscribed {
				t.logger.Trace("re-subscribing resource for stream", "resource", name)
			} else {
				t.logger.Trace("subscribing resource for stream", "resource", name)
			}
		}

		for _, name := range req.ResourceNamesUnsubscribe {
			if _, ok := t.subscriptions[name]; !ok {
				continue
			}
			delete(t.subscriptions, name)
			t.logger.Trace("unsubscribing resource for stream", "resource", name)
			// NOTE: we'll let the normal differential comparison handle cleaning up resourceVersions
		}
	}

	if registeredThisTime {
		return deltaRecvNewSubscription
	}
	return deltaRecvResponseAck
}

func (t *xDSDeltaType) ack(nonce string) {
	pending, ok := t.pendingUpdates[nonce]
	if !ok {
		return
	}

	for name, obj := range pending {
		if obj.Remove {
			delete(t.resourceVersions, name)
			continue
		}

		t.resourceVersions[name] = obj.Version
	}
	t.sentToEnvoyOnce = true
	delete(t.pendingUpdates, nonce)
}

func (t *xDSDeltaType) nack(nonce string) {
	delete(t.pendingUpdates, nonce)
}

func (t *xDSDeltaType) SendIfNew(
	currentVersions map[string]string, // type => name => version (as consul knows right now)
	resourceMap *xdscommon.IndexedResources,
	nonce *uint64,
	upsert, remove bool,
) (error, bool) {
	if t == nil || !t.registered {
		return nil, false
	}

	// Wait for Envoy to catch up with this delta type before sending something new.
	if len(t.pendingUpdates) > 0 {
		return nil, false
	}

	logger := t.logger.With("typeUrl", t.typeURL)

	allowEmpty := t.allowEmptyFn != nil && t.allowEmptyFn()

	// Zero length resource responses should be ignored and are the result of no
	// data yet. Notice that this caused a bug originally where we had zero
	// healthy endpoints for an upstream that would cause Envoy to hang waiting
	// for the EDS response. This is fixed though by ensuring we send an explicit
	// empty LoadAssignment resource for the cluster rather than allowing junky
	// empty resources.
	if len(currentVersions) == 0 && !allowEmpty {
		// Nothing to send yet
		return nil, false
	}

	resp, updates, err := t.createDeltaResponse(currentVersions, resourceMap, upsert, remove)
	if err != nil {
		return err, false
	}

	if resp == nil {
		return nil, false
	}

	*nonce++
	resp.Nonce = fmt.Sprintf("%08x", *nonce)

	logTraceResponse(t.logger, "Incremental xDS v3", resp)

	logger.Trace("sending response", "nonce", resp.Nonce)
	if err := t.stream.Send(resp); err != nil {
		return err, false
	}
	logger.Trace("sent response", "nonce", resp.Nonce)

	// Certain xDS types are children of other types, meaning that if an update is pushed for a parent,
	// we MUST send new data for all its children. Envoy will NOT re-subscribe to the child data upon
	// receiving updates for the parent, so we need to handle this ourselves.
	//
	// Note that we do not check whether the deltaChild.childType is registered here, since we send
	// parent types before child types, meaning that it's expected on first send of a parent that
	// there are no subscriptions for the child type.
	if t.deltaChild != nil {
		for name := range updates {
			if children, ok := resourceMap.ChildIndex[t.typeURL][name]; ok {
				// Capture the relevant child resource names on this pending update so
				// we can know the linked children if Envoy ever re-subscribes to the parent resource.
				t.deltaChild.childrenNames[name] = children

				for _, childName := range children {
					t.ensureChildResend(name, childName)
				}
			}
		}
	}
	t.pendingUpdates[resp.Nonce] = updates

	return nil, true
}

func (t *xDSDeltaType) createDeltaResponse(
	currentVersions map[string]string, // name => version (as consul knows right now)
	resourceMap *xdscommon.IndexedResources,
	upsert, remove bool,
) (*envoy_discovery_v3.DeltaDiscoveryResponse, map[string]PendingUpdate, error) {
	// compute difference
	var (
		hasRelevantUpdates = false
		updates            = make(map[string]PendingUpdate)
	)

	if t.wildcard {
		// First find things that need updating or deleting
		for name, envoyVers := range t.resourceVersions {
			currVers, ok := currentVersions[name]
			if !ok {
				if remove {
					hasRelevantUpdates = true
				}
				updates[name] = PendingUpdate{Remove: true}
			} else if currVers != envoyVers {
				if upsert {
					hasRelevantUpdates = true
				}
				updates[name] = PendingUpdate{Version: currVers}
			}
		}

		// Now find new things
		for name, currVers := range currentVersions {
			if _, known := t.resourceVersions[name]; known {
				continue
			}
			if upsert {
				hasRelevantUpdates = true
			}
			updates[name] = PendingUpdate{Version: currVers}
		}
	} else {
		// First find things that need updating or deleting

		// Walk the list of things currently stored in envoy
		for name, envoyVers := range t.resourceVersions {
			if t.subscribed(name) {
				if currVers, ok := currentVersions[name]; ok {
					if currVers != envoyVers {
						if upsert {
							hasRelevantUpdates = true
						}
						updates[name] = PendingUpdate{Version: currVers}
					}
				}
			}
		}

		// Now find new things not in envoy yet
		for name := range t.subscriptions {
			if _, known := t.resourceVersions[name]; known {
				continue
			}
			if currVers, ok := currentVersions[name]; ok {
				updates[name] = PendingUpdate{Version: currVers}
				if upsert {
					hasRelevantUpdates = true
				}
			}
		}
	}

	if !hasRelevantUpdates && t.sentToEnvoyOnce {
		return nil, nil, nil
	}

	// now turn this into a disco response
	resp := &envoy_discovery_v3.DeltaDiscoveryResponse{
		// TODO(rb): consider putting something in SystemVersionInfo?
		TypeUrl: t.typeURL,
	}
	realUpdates := make(map[string]PendingUpdate)
	for name, obj := range updates {
		if obj.Remove {
			if remove {
				resp.RemovedResources = append(resp.RemovedResources, name)
				realUpdates[name] = PendingUpdate{Remove: true}
			}
		} else if upsert {
			resources, ok := resourceMap.Index[t.typeURL]
			if !ok {
				return nil, nil, fmt.Errorf("unknown type url: %s", t.typeURL)
			}
			res, ok := resources[name]
			if !ok {
				return nil, nil, fmt.Errorf("unknown name for type url %q: %s", t.typeURL, name)
			}
			any, err := anypb.New(res)
			if err != nil {
				return nil, nil, err
			}

			resp.Resources = append(resp.Resources, &envoy_discovery_v3.Resource{
				Name:     name,
				Resource: any,
				Version:  obj.Version,
			})
			realUpdates[name] = obj
		}
	}

	return resp, realUpdates, nil
}

func (t *xDSDeltaType) ensureChildResend(parentName, childName string) {
	if !t.subscribed(childName) {
		return
	}
	t.logger.Trace(
		"triggering implicit update of resource",
		"typeUrl", t.typeURL,
		"resource", parentName,
		"childTypeUrl", t.deltaChild.childType.typeURL,
		"childResource", childName,
	)
	// resourceVersions tracks the last known version for this childName that Envoy
	// has ACKed. By setting this to empty it effectively tells us that Envoy does
	// not have any data for that child, and we need to re-send.
	if _, exist := t.deltaChild.childType.resourceVersions[childName]; exist {
		t.deltaChild.childType.resourceVersions[childName] = ""
	}

	if xdsProtocolLegacyChildResend {
		return
		// TODO: This legacy behavior can be removed in 1.19, provided there are no outstanding issues.
		//
		// In this legacy mode, there is a confirmed race condition:
		// - Send update endpoints
		// - Send update cluster
		// - Recv ACK endpoints
		// - Recv ACK cluster
		//
		// When this situation happens, Envoy wipes the child endpoints when the cluster is updated,
		// but it would never receive new ones. The endpoints would not be resent, because their hash
		// never changed since the previous ACK.
		//
		// Due to ambiguity with the Envoy protocol [https://github.com/envoyproxy/envoy/issues/13009],
		// it's difficult to state with certainty that no other unexpected side-effects are possible.
		// This legacy escape hatch is left in-place in case some other complex race condition crops up.
		//
		// Longer-term, we should modify the hash of children to include the parent hash so that this
		// behavior is implicitly handled, rather than being an edge case.
	}

	// pendingUpdates can contain newer versions that have been sent to Envoy but
	// that we haven't processed an ACK for yet. These need to be cleared out, too,
	// so that they aren't moved to resourceVersions by ack()
	for nonce := range t.deltaChild.childType.pendingUpdates {
		delete(t.deltaChild.childType.pendingUpdates[nonce], childName)
	}
}

func computeResourceVersions(resourceMap *xdscommon.IndexedResources) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string)
	for typeUrl, resources := range resourceMap.Index {
		m, err := hashResourceMap(resources)
		if err != nil {
			return nil, fmt.Errorf("failed to hash resources for %q: %v", typeUrl, err)
		}
		out[typeUrl] = m
	}
	return out, nil
}

func populateChildIndexMap(resourceMap *xdscommon.IndexedResources) error {
	// LDS and RDS have a more complicated relationship.
	for name, res := range resourceMap.Index[xdscommon.ListenerType] {
		listener := res.(*envoy_listener_v3.Listener)
		rdsRouteNames, err := extractRdsResourceNames(listener)
		if err != nil {
			return err
		}
		resourceMap.ChildIndex[xdscommon.ListenerType][name] = rdsRouteNames
	}

	// CDS and EDS share exact names.
	for name := range resourceMap.Index[xdscommon.ClusterType] {
		resourceMap.ChildIndex[xdscommon.ClusterType][name] = []string{name}
	}

	return nil
}

func hashResourceMap(resources map[string]proto.Message) (map[string]string, error) {
	m := make(map[string]string)
	for name, res := range resources {
		h, err := hashResource(res)
		if err != nil {
			return nil, fmt.Errorf("failed to hash resource %q: %v", name, err)
		}
		m[name] = h
	}
	return m, nil
}

// hashResource will take a resource and create a SHA256 hash sum out of the marshaled bytes
func hashResource(res proto.Message) (string, error) {
	h := sha256.New()
	marshaller := proto.MarshalOptions{Deterministic: true}

	data, err := marshaller.Marshal(res)
	if err != nil {
		return "", err
	}
	h.Write(data)

	return hex.EncodeToString(h.Sum(nil)), nil
}
