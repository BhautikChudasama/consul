// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: pbresource/resource.proto

package pbresource

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ResourceServiceClient is the client API for ResourceService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ResourceServiceClient interface {
	// Read a resource by ID.
	//
	// By default, reads are eventually consistent, but you can opt-in to strong
	// consistency via the x-consul-consistency-mode metadata (see ResourceService
	// docs for more info).
	//
	// Errors with NotFound if the resource is not found.
	//
	// Errors with InvalidArgument if the request fails validation or the resource
	// is stored as a type with a different GroupVersion than was requested.
	//
	// Errors with PermissionDenied if the caller is not authorized to read
	// the resource.
	Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error)
	// Write a resource.
	//
	// To perform a CAS (Compare-And-Swap) write, provide the current resource
	// version in the Resource.Version field. If the given version doesn't match
	// what is currently stored, an Aborted error code will be returned.
	//
	// To perform a blanket write (update regardless of the stored version),
	// provide an empty Version in the Resource.Version field. Note that the
	// write may still fail due to not being able to internally do a CAS write
	// and return an Aborted error code.
	//
	// Resource.Id.Uid can (and by controllers, should) be provided to avoid
	// accidentally modifying a resource if it has been deleted and recreated.
	// If the given Uid doesn't match what is stored, a FailedPrecondition error
	// code will be returned.
	//
	// It is not possible to modify the resource's status using Write. You must
	// use WriteStatus instead.
	Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error)
	// WriteStatus updates one of the resource's statuses. It should only be used
	// by controllers.
	//
	// To perform a CAS (Compare-And-Swap) write, provide the current resource
	// version in the Version field. If the given version doesn't match what is
	// currently stored, an Aborted error code will be returned.
	//
	// Note: in most cases, CAS status updates are not necessary because updates
	// are scoped to a specific status key and controllers are leader-elected so
	// there is no chance of a conflict.
	//
	// Id.Uid must be provided to avoid accidentally modifying a resource if it has
	// been deleted and recreated. If the given Uid doesn't match what is stored,
	// a FailedPrecondition error code will be returned.
	WriteStatus(ctx context.Context, in *WriteStatusRequest, opts ...grpc.CallOption) (*WriteStatusResponse, error)
	// List resources of a given type, tenancy, and optionally name prefix.
	//
	// To list resources across all tenancy units, provide the wildcard "*" value.
	//
	// Results are eventually consistent (see ResourceService docs for more info).
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error)
	// List resources of a given owner.
	//
	// Results are eventually consistent (see ResourceService docs for more info).
	ListByOwner(ctx context.Context, in *ListByOwnerRequest, opts ...grpc.CallOption) (*ListByOwnerResponse, error)
	// Delete a resource by ID.
	//
	// Deleting a non-existent resource will return a successful response for
	// idempotency.
	//
	// To perform a CAS (Compare-And-Swap) deletion, provide the current resource
	// version in the Version field. If the given version doesn't match what is
	// currently stored, an Aborted error code will be returned.
	//
	// Resource.Id.Uid can (and by controllers, should) be provided to avoid
	// accidentally modifying a resource if it has been deleted and recreated.
	// If the given Uid doesn't match what is stored, a FailedPrecondition error
	// code will be returned.
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	// WatchList watches resources of the given type, tenancy, and optionally name
	// prefix. It returns results for the current state-of-the-world at the start
	// of the stream, and delta events whenever resources are written or deleted.
	//
	// To watch resources across all tenancy units, provide the wildcard "*" value.
	//
	// WatchList makes no guarantees about event timeliness (e.g. an event for a
	// write may not be received immediately), but it does guarantee that events
	// will be emitted in the correct order. See ResourceService docs for more
	// info about consistency guarentees.
	//
	// buf:lint:ignore RPC_RESPONSE_STANDARD_NAME
	WatchList(ctx context.Context, in *WatchListRequest, opts ...grpc.CallOption) (ResourceService_WatchListClient, error)
}

type resourceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewResourceServiceClient(cc grpc.ClientConnInterface) ResourceServiceClient {
	return &resourceServiceClient{cc}
}

func (c *resourceServiceClient) Read(ctx context.Context, in *ReadRequest, opts ...grpc.CallOption) (*ReadResponse, error) {
	out := new(ReadResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/Read", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) Write(ctx context.Context, in *WriteRequest, opts ...grpc.CallOption) (*WriteResponse, error) {
	out := new(WriteResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/Write", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) WriteStatus(ctx context.Context, in *WriteStatusRequest, opts ...grpc.CallOption) (*WriteStatusResponse, error) {
	out := new(WriteStatusResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/WriteStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error) {
	out := new(ListResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) ListByOwner(ctx context.Context, in *ListByOwnerRequest, opts ...grpc.CallOption) (*ListByOwnerResponse, error) {
	out := new(ListByOwnerResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/ListByOwner", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, "/hashicorp.consul.resource.ResourceService/Delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *resourceServiceClient) WatchList(ctx context.Context, in *WatchListRequest, opts ...grpc.CallOption) (ResourceService_WatchListClient, error) {
	stream, err := c.cc.NewStream(ctx, &ResourceService_ServiceDesc.Streams[0], "/hashicorp.consul.resource.ResourceService/WatchList", opts...)
	if err != nil {
		return nil, err
	}
	x := &resourceServiceWatchListClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ResourceService_WatchListClient interface {
	Recv() (*WatchEvent, error)
	grpc.ClientStream
}

type resourceServiceWatchListClient struct {
	grpc.ClientStream
}

func (x *resourceServiceWatchListClient) Recv() (*WatchEvent, error) {
	m := new(WatchEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ResourceServiceServer is the server API for ResourceService service.
// All implementations should embed UnimplementedResourceServiceServer
// for forward compatibility
type ResourceServiceServer interface {
	// Read a resource by ID.
	//
	// By default, reads are eventually consistent, but you can opt-in to strong
	// consistency via the x-consul-consistency-mode metadata (see ResourceService
	// docs for more info).
	//
	// Errors with NotFound if the resource is not found.
	//
	// Errors with InvalidArgument if the request fails validation or the resource
	// is stored as a type with a different GroupVersion than was requested.
	//
	// Errors with PermissionDenied if the caller is not authorized to read
	// the resource.
	Read(context.Context, *ReadRequest) (*ReadResponse, error)
	// Write a resource.
	//
	// To perform a CAS (Compare-And-Swap) write, provide the current resource
	// version in the Resource.Version field. If the given version doesn't match
	// what is currently stored, an Aborted error code will be returned.
	//
	// To perform a blanket write (update regardless of the stored version),
	// provide an empty Version in the Resource.Version field. Note that the
	// write may still fail due to not being able to internally do a CAS write
	// and return an Aborted error code.
	//
	// Resource.Id.Uid can (and by controllers, should) be provided to avoid
	// accidentally modifying a resource if it has been deleted and recreated.
	// If the given Uid doesn't match what is stored, a FailedPrecondition error
	// code will be returned.
	//
	// It is not possible to modify the resource's status using Write. You must
	// use WriteStatus instead.
	Write(context.Context, *WriteRequest) (*WriteResponse, error)
	// WriteStatus updates one of the resource's statuses. It should only be used
	// by controllers.
	//
	// To perform a CAS (Compare-And-Swap) write, provide the current resource
	// version in the Version field. If the given version doesn't match what is
	// currently stored, an Aborted error code will be returned.
	//
	// Note: in most cases, CAS status updates are not necessary because updates
	// are scoped to a specific status key and controllers are leader-elected so
	// there is no chance of a conflict.
	//
	// Id.Uid must be provided to avoid accidentally modifying a resource if it has
	// been deleted and recreated. If the given Uid doesn't match what is stored,
	// a FailedPrecondition error code will be returned.
	WriteStatus(context.Context, *WriteStatusRequest) (*WriteStatusResponse, error)
	// List resources of a given type, tenancy, and optionally name prefix.
	//
	// To list resources across all tenancy units, provide the wildcard "*" value.
	//
	// Results are eventually consistent (see ResourceService docs for more info).
	List(context.Context, *ListRequest) (*ListResponse, error)
	// List resources of a given owner.
	//
	// Results are eventually consistent (see ResourceService docs for more info).
	ListByOwner(context.Context, *ListByOwnerRequest) (*ListByOwnerResponse, error)
	// Delete a resource by ID.
	//
	// Deleting a non-existent resource will return a successful response for
	// idempotency.
	//
	// To perform a CAS (Compare-And-Swap) deletion, provide the current resource
	// version in the Version field. If the given version doesn't match what is
	// currently stored, an Aborted error code will be returned.
	//
	// Resource.Id.Uid can (and by controllers, should) be provided to avoid
	// accidentally modifying a resource if it has been deleted and recreated.
	// If the given Uid doesn't match what is stored, a FailedPrecondition error
	// code will be returned.
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	// WatchList watches resources of the given type, tenancy, and optionally name
	// prefix. It returns results for the current state-of-the-world at the start
	// of the stream, and delta events whenever resources are written or deleted.
	//
	// To watch resources across all tenancy units, provide the wildcard "*" value.
	//
	// WatchList makes no guarantees about event timeliness (e.g. an event for a
	// write may not be received immediately), but it does guarantee that events
	// will be emitted in the correct order. See ResourceService docs for more
	// info about consistency guarentees.
	//
	// buf:lint:ignore RPC_RESPONSE_STANDARD_NAME
	WatchList(*WatchListRequest, ResourceService_WatchListServer) error
}

// UnimplementedResourceServiceServer should be embedded to have forward compatible implementations.
type UnimplementedResourceServiceServer struct {
}

func (UnimplementedResourceServiceServer) Read(context.Context, *ReadRequest) (*ReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedResourceServiceServer) Write(context.Context, *WriteRequest) (*WriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedResourceServiceServer) WriteStatus(context.Context, *WriteStatusRequest) (*WriteStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteStatus not implemented")
}
func (UnimplementedResourceServiceServer) List(context.Context, *ListRequest) (*ListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedResourceServiceServer) ListByOwner(context.Context, *ListByOwnerRequest) (*ListByOwnerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListByOwner not implemented")
}
func (UnimplementedResourceServiceServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedResourceServiceServer) WatchList(*WatchListRequest, ResourceService_WatchListServer) error {
	return status.Errorf(codes.Unimplemented, "method WatchList not implemented")
}

// UnsafeResourceServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ResourceServiceServer will
// result in compilation errors.
type UnsafeResourceServiceServer interface {
	mustEmbedUnimplementedResourceServiceServer()
}

func RegisterResourceServiceServer(s grpc.ServiceRegistrar, srv ResourceServiceServer) {
	s.RegisterService(&ResourceService_ServiceDesc, srv)
}

func _ResourceService_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/Read",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).Read(ctx, req.(*ReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_Write_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).Write(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/Write",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).Write(ctx, req.(*WriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_WriteStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).WriteStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/WriteStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).WriteStatus(ctx, req.(*WriteStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).List(ctx, req.(*ListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_ListByOwner_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListByOwnerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).ListByOwner(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/ListByOwner",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).ListByOwner(ctx, req.(*ListByOwnerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ResourceServiceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hashicorp.consul.resource.ResourceService/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ResourceServiceServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ResourceService_WatchList_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(WatchListRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ResourceServiceServer).WatchList(m, &resourceServiceWatchListServer{stream})
}

type ResourceService_WatchListServer interface {
	Send(*WatchEvent) error
	grpc.ServerStream
}

type resourceServiceWatchListServer struct {
	grpc.ServerStream
}

func (x *resourceServiceWatchListServer) Send(m *WatchEvent) error {
	return x.ServerStream.SendMsg(m)
}

// ResourceService_ServiceDesc is the grpc.ServiceDesc for ResourceService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ResourceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hashicorp.consul.resource.ResourceService",
	HandlerType: (*ResourceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Read",
			Handler:    _ResourceService_Read_Handler,
		},
		{
			MethodName: "Write",
			Handler:    _ResourceService_Write_Handler,
		},
		{
			MethodName: "WriteStatus",
			Handler:    _ResourceService_WriteStatus_Handler,
		},
		{
			MethodName: "List",
			Handler:    _ResourceService_List_Handler,
		},
		{
			MethodName: "ListByOwner",
			Handler:    _ResourceService_ListByOwner_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _ResourceService_Delete_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "WatchList",
			Handler:       _ResourceService_WatchList_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "pbresource/resource.proto",
}
