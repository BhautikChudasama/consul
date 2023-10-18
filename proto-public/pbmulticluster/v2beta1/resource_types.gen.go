// Code generated by protoc-gen-resource-types. DO NOT EDIT.

package multiclusterv2beta1

import (
	"github.com/hashicorp/consul/proto-public/pbresource"
)

const (
	GroupName = "multicluster"
	Version   = "v2beta1"
)

/* ---------------------------------------------------------------------------
 * hashicorp.consul.multicluster.v2beta1.ComputedExportedServices
 *
 * This following section contains constants variables and utility methods
 * for interacting with this kind of resource.
 * -------------------------------------------------------------------------*/

const ComputedExportedServicesKind = "ComputedExportedServices"

var ComputedExportedServicesType = &pbresource.Type{
	Group:        GroupName,
	GroupVersion: Version,
	Kind:         ComputedExportedServicesKind,
}

func (_ *ComputedExportedServices) GetResourceType() *pbresource.Type {
	return ComputedExportedServicesType
}

/* ---------------------------------------------------------------------------
 * hashicorp.consul.multicluster.v2beta1.ExportedServices
 *
 * This following section contains constants variables and utility methods
 * for interacting with this kind of resource.
 * -------------------------------------------------------------------------*/

const ExportedServicesKind = "ExportedServices"

var ExportedServicesType = &pbresource.Type{
	Group:        GroupName,
	GroupVersion: Version,
	Kind:         ExportedServicesKind,
}

func (_ *ExportedServices) GetResourceType() *pbresource.Type {
	return ExportedServicesType
}

/* ---------------------------------------------------------------------------
 * hashicorp.consul.multicluster.v2beta1.NamespaceExportedServices
 *
 * This following section contains constants variables and utility methods
 * for interacting with this kind of resource.
 * -------------------------------------------------------------------------*/

const NamespaceExportedServicesKind = "NamespaceExportedServices"

var NamespaceExportedServicesType = &pbresource.Type{
	Group:        GroupName,
	GroupVersion: Version,
	Kind:         NamespaceExportedServicesKind,
}

func (_ *NamespaceExportedServices) GetResourceType() *pbresource.Type {
	return NamespaceExportedServicesType
}

/* ---------------------------------------------------------------------------
 * hashicorp.consul.multicluster.v2beta1.PartitionExportedServices
 *
 * This following section contains constants variables and utility methods
 * for interacting with this kind of resource.
 * -------------------------------------------------------------------------*/

const PartitionExportedServicesKind = "PartitionExportedServices"

var PartitionExportedServicesType = &pbresource.Type{
	Group:        GroupName,
	GroupVersion: Version,
	Kind:         PartitionExportedServicesKind,
}

func (_ *PartitionExportedServices) GetResourceType() *pbresource.Type {
	return PartitionExportedServicesType
}
