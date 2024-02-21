// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package routes

import (
	"fmt"

	"github.com/hashicorp/consul/internal/mesh/internal/types"
	"github.com/hashicorp/consul/internal/resource"
	pbcatalog "github.com/hashicorp/consul/proto-public/pbcatalog/v2beta1"
	pbmesh "github.com/hashicorp/consul/proto-public/pbmesh/v2beta1"
	"github.com/hashicorp/consul/proto-public/pbresource"
)

// RelatedResources is a spiritual successor of *configentry.DiscoveryChainSet
type RelatedResources struct {
	// ComputedIDList will be a list of ComputedRoutes IDs that were requested in the case of EW traffic and
	// a list of ComputedAPIGateway IDs in the case of NS traffic
	ComputedIDList []*pbresource.ID
	// RoutesByParentRef is a map of a parent Service to the xRoutes that compose it.
	RoutesByParentRef        map[resource.ReferenceKey]map[resource.ReferenceKey]struct{}
	HTTPRoutes               map[resource.ReferenceKey]*types.DecodedHTTPRoute
	GRPCRoutes               map[resource.ReferenceKey]*types.DecodedGRPCRoute
	TCPRoutes                map[resource.ReferenceKey]*types.DecodedTCPRoute
	Services                 map[resource.ReferenceKey]*types.DecodedService
	APIGateways              map[resource.ReferenceKey]*types.DecodedAPIGateway
	ComputedFailoverPolicies map[resource.ReferenceKey]*types.DecodedComputedFailoverPolicy
	DestinationPolicies      map[resource.ReferenceKey]*types.DecodedDestinationPolicy
}

func NewRelatedResources() *RelatedResources {
	return &RelatedResources{
		RoutesByParentRef:        make(map[resource.ReferenceKey]map[resource.ReferenceKey]struct{}),
		HTTPRoutes:               make(map[resource.ReferenceKey]*types.DecodedHTTPRoute),
		GRPCRoutes:               make(map[resource.ReferenceKey]*types.DecodedGRPCRoute),
		TCPRoutes:                make(map[resource.ReferenceKey]*types.DecodedTCPRoute),
		Services:                 make(map[resource.ReferenceKey]*types.DecodedService),
		APIGateways:              make(map[resource.ReferenceKey]*types.DecodedAPIGateway),
		ComputedFailoverPolicies: make(map[resource.ReferenceKey]*types.DecodedComputedFailoverPolicy),
		DestinationPolicies:      make(map[resource.ReferenceKey]*types.DecodedDestinationPolicy),
	}
}

func (r *RelatedResources) AddComputedRoutesIDs(list ...*pbresource.ID) *RelatedResources {
	for _, id := range list {
		r.AddComputedRoutesID(id)
	}
	return r
}

func (r *RelatedResources) AddComputedRoutesID(id *pbresource.ID) *RelatedResources {
	assertResourceTypeIn([]*pbresource.Type{pbmesh.ComputedRoutesType, pbmesh.ComputedGatewayConfigurationType}, id.Type)
	r.ComputedIDList = append(r.ComputedIDList, id)
	return r
}

// AddResources must only be called with valid *resource.DecodedResource[T]
// types.
//
// This is provided as a testing convenience. Non-test code should call the
// type-specific adder.
func (r *RelatedResources) AddResources(list ...any) *RelatedResources {
	for _, res := range list {
		r.AddResource(res)
	}
	return r
}

// AddResource must only be called with valid *resource.DecodedResource[T] types.
//
// This is provided as a testing convenience. Non-test code should call the
// type-specific adder.
func (r *RelatedResources) AddResource(res any) {
	if res == nil {
		return
	}

	switch dec := res.(type) {
	case *types.DecodedHTTPRoute:
		r.AddHTTPRoute(dec)
	case *types.DecodedGRPCRoute:
		r.AddGRPCRoute(dec)
	case *types.DecodedTCPRoute:
		r.AddTCPRoute(dec)
	case *types.DecodedDestinationPolicy:
		r.AddDestinationPolicy(dec)
	case *types.DecodedService:
		r.AddService(dec)
	case *types.DecodedAPIGateway:
		r.AddAPIGateway(dec)
	case *types.DecodedComputedFailoverPolicy:
		r.AddComputedFailoverPolicy(dec)
	default:
		panic(fmt.Sprintf("unknown decoded resource type: %T", res))
	}
}

func (r *RelatedResources) AddHTTPRoute(dec *types.DecodedHTTPRoute) {
	assertResourceType(pbmesh.HTTPRouteType, dec.Id.Type)
	r.addRouteSetEntries(dec.Resource, dec.Data)
	addResource(dec.Resource.Id, dec, r.HTTPRoutes)
}

func (r *RelatedResources) AddGRPCRoute(dec *types.DecodedGRPCRoute) {
	assertResourceType(pbmesh.GRPCRouteType, dec.Id.Type)
	r.addRouteSetEntries(dec.Resource, dec.Data)
	addResource(dec.Resource.Id, dec, r.GRPCRoutes)
}

func (r *RelatedResources) AddTCPRoute(dec *types.DecodedTCPRoute) {
	assertResourceType(pbmesh.TCPRouteType, dec.Id.Type)
	r.addRouteSetEntries(dec.Resource, dec.Data)
	addResource(dec.Resource.Id, dec, r.TCPRoutes)
}

func (r *RelatedResources) AddDestinationPolicy(dec *types.DecodedDestinationPolicy) {
	assertResourceType(pbmesh.DestinationPolicyType, dec.Id.Type)
	addResource(dec.Resource.Id, dec, r.DestinationPolicies)
}

func (r *RelatedResources) AddService(dec *types.DecodedService) {
	assertResourceType(pbcatalog.ServiceType, dec.Id.Type)
	addResource(dec.Resource.Id, dec, r.Services)
}

func (r *RelatedResources) AddAPIGateway(dec *types.DecodedAPIGateway) {
	assertResourceType(pbmesh.APIGatewayType, dec.Id.Type)
	addResource(dec.Resource.Id, dec, r.APIGateways)
}

func (r *RelatedResources) AddComputedFailoverPolicy(dec *types.DecodedComputedFailoverPolicy) {
	assertResourceType(pbcatalog.ComputedFailoverPolicyType, dec.Id.Type)
	addResource(dec.Resource.Id, dec, r.ComputedFailoverPolicies)
}

func (r *RelatedResources) addRouteSetEntries(
	res *pbresource.Resource,
	xroute types.XRouteData,
) {
	if res == nil || xroute == nil {
		return
	}

	routeRK := resource.NewReferenceKey(res.Id)

	for _, parentRef := range xroute.GetParentRefs() {
		if parentRef.Ref == nil || (!types.IsServiceType(parentRef.Ref.Type) && !types.IsAPIGatewayType(parentRef.Ref.Type)) {
			continue
		}
		svcRK := resource.NewReferenceKey(parentRef.Ref)

		r.addRouteByParentRef(svcRK, routeRK)
	}
}

func (r *RelatedResources) addRouteByParentRef(svcRK, xRouteRK resource.ReferenceKey) {
	m, ok := r.RoutesByParentRef[svcRK]
	if !ok {
		m = make(map[resource.ReferenceKey]struct{})
		r.RoutesByParentRef[svcRK] = m
	}
	m[xRouteRK] = struct{}{}
}

type RouteWalkFunc func(
	rk resource.ReferenceKey,
	res *pbresource.Resource,
	route types.XRouteData,
)

func (r *RelatedResources) WalkRoutes(fn RouteWalkFunc) {
	for rk, route := range r.HTTPRoutes {
		fn(rk, route.Resource, route.Data)
	}
	for rk, route := range r.GRPCRoutes {
		fn(rk, route.Resource, route.Data)
	}
	for rk, route := range r.TCPRoutes {
		fn(rk, route.Resource, route.Data)
	}
}

func (r *RelatedResources) WalkRoutesForParentRef(parentRef *pbresource.Reference, fn RouteWalkFunc) {
	assertResourceType(pbcatalog.ServiceType, parentRef.Type)
	routeMap := r.RoutesByParentRef[resource.NewReferenceKey(parentRef)]
	if len(routeMap) == 0 {
		return
	}

	for rk := range routeMap {
		if route, ok := r.HTTPRoutes[rk]; ok {
			fn(rk, route.Resource, route.Data)
			continue
		}
		if route, ok := r.GRPCRoutes[rk]; ok {
			fn(rk, route.Resource, route.Data)
			continue
		}
		if route, ok := r.TCPRoutes[rk]; ok {
			fn(rk, route.Resource, route.Data)
			continue
		}
	}
}

func (r *RelatedResources) GetService(ref resource.ReferenceOrID) *types.DecodedService {
	assertResourceType(pbcatalog.ServiceType, ref.GetType())
	return r.Services[resource.NewReferenceKey(ref)]
}

func (r *RelatedResources) GetComputedFailoverPolicy(ref resource.ReferenceOrID) *types.DecodedComputedFailoverPolicy {
	assertResourceType(pbcatalog.ComputedFailoverPolicyType, ref.GetType())
	return r.ComputedFailoverPolicies[resource.NewReferenceKey(ref)]
}

func (r *RelatedResources) GetComputedFailoverPolicyForService(ref resource.ReferenceOrID) *types.DecodedComputedFailoverPolicy {
	failRef := &pbresource.Reference{
		Type:    pbcatalog.ComputedFailoverPolicyType,
		Tenancy: ref.GetTenancy(),
		Name:    ref.GetName(),
	}
	return r.GetComputedFailoverPolicy(failRef)
}

func (r *RelatedResources) GetDestinationPolicy(ref resource.ReferenceOrID) *types.DecodedDestinationPolicy {
	assertResourceType(pbmesh.DestinationPolicyType, ref.GetType())
	return r.DestinationPolicies[resource.NewReferenceKey(ref)]
}

func (r *RelatedResources) GetDestinationPolicyForService(ref resource.ReferenceOrID) *types.DecodedDestinationPolicy {
	destRef := &pbresource.Reference{
		Type:    pbmesh.DestinationPolicyType,
		Tenancy: ref.GetTenancy(),
		Name:    ref.GetName(),
	}
	return r.GetDestinationPolicy(destRef)
}

func addResource[V any](id *pbresource.ID, res *V, m map[resource.ReferenceKey]*V) {
	if res == nil {
		return
	}

	rk := resource.NewReferenceKey(id)
	if _, ok := m[rk]; !ok {
		m[rk] = res
	}
}
