// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"encoding/json"
)

// MeshConfigEntry manages the global configuration for all service mesh
// proxies.
type MeshConfigEntry struct {
	// Partition is the partition the MeshConfigEntry applies to.
	// Partitioning is a Consul Enterprise feature.
	Partition string `json:",omitempty"`

	// Namespace is the namespace the MeshConfigEntry applies to.
	// Namespacing is a Consul Enterprise feature.
	Namespace string `json:",omitempty"`

	// TransparentProxy applies configuration specific to proxies
	// in transparent mode.
	TransparentProxy TransparentProxyMeshConfig `alias:"transparent_proxy"`

	// ValidateClusters controls whether the clusters the route table refers to are validated. The default value is
	// false. When set to false and a route refers to a cluster that does not exist, the route table loads and routing
	// to a non-existent cluster results in a 404. When set to true and the route is set to a cluster that do not exist,
	// the route table will not load. For more information, refer to
	// [HTTP route configuration in the Envoy docs](https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/route/v3/route.proto#envoy-v3-api-field-config-route-v3-routeconfiguration-validate-clusters)
	// for more details.
	ValidateClusters bool `json:",omitempty" alias:"validate_clusters"`

	TLS *MeshTLSConfig `json:",omitempty"`

	HTTP *MeshHTTPConfig `json:",omitempty"`

	Peering *PeeringMeshConfig `json:",omitempty"`

	Meta map[string]string `json:",omitempty"`

	// CreateIndex is the Raft index this entry was created at. This is a
	// read-only field.
	CreateIndex uint64

	// ModifyIndex is used for the Check-And-Set operations and can also be fed
	// back into the WaitIndex of the QueryOptions in order to perform blocking
	// queries.
	ModifyIndex uint64
}

type TransparentProxyMeshConfig struct {
	MeshDestinationsOnly bool `alias:"mesh_destinations_only"`
}

type MeshTLSConfig struct {
	Incoming *MeshDirectionalTLSConfig `json:",omitempty"`
	Outgoing *MeshDirectionalTLSConfig `json:",omitempty"`
}

type MeshDirectionalTLSConfig struct {
	TLSMinVersion string   `json:",omitempty" alias:"tls_min_version"`
	TLSMaxVersion string   `json:",omitempty" alias:"tls_max_version"`
	CipherSuites  []string `json:",omitempty" alias:"cipher_suites"`
}

type MeshHTTPConfig struct {
	SanitizeXForwardedClientCert bool `alias:"sanitize_x_forwarded_client_cert"`
}

type PeeringMeshConfig struct {
	PeerThroughMeshGateways bool `json:",omitempty" alias:"peer_through_mesh_gateways"`
}

func (e *MeshConfigEntry) GetKind() string            { return MeshConfig }
func (e *MeshConfigEntry) GetName() string            { return MeshConfigMesh }
func (e *MeshConfigEntry) GetPartition() string       { return e.Partition }
func (e *MeshConfigEntry) GetNamespace() string       { return e.Namespace }
func (e *MeshConfigEntry) GetMeta() map[string]string { return e.Meta }
func (e *MeshConfigEntry) GetCreateIndex() uint64     { return e.CreateIndex }
func (e *MeshConfigEntry) GetModifyIndex() uint64     { return e.ModifyIndex }

// MarshalJSON adds the Kind field so that the JSON can be decoded back into the
// correct type.
func (e *MeshConfigEntry) MarshalJSON() ([]byte, error) {
	type Alias MeshConfigEntry
	source := &struct {
		Kind string
		*Alias
	}{
		Kind:  MeshConfig,
		Alias: (*Alias)(e),
	}
	return json.Marshal(source)
}
