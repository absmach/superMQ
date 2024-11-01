// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.28.2
// source: domains/v1/domains.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	DomainsService_DeleteUserFromDomains_FullMethodName = "/domains.v1.DomainsService/DeleteUserFromDomains"
)

// DomainsServiceClient is the client API for DomainsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// DomainsService is a service that provides access to domains
// functionalities for magistrala services.
type DomainsServiceClient interface {
	DeleteUserFromDomains(ctx context.Context, in *DeleteUserReq, opts ...grpc.CallOption) (*DeleteUserRes, error)
}

type domainsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDomainsServiceClient(cc grpc.ClientConnInterface) DomainsServiceClient {
	return &domainsServiceClient{cc}
}

func (c *domainsServiceClient) DeleteUserFromDomains(ctx context.Context, in *DeleteUserReq, opts ...grpc.CallOption) (*DeleteUserRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteUserRes)
	err := c.cc.Invoke(ctx, DomainsService_DeleteUserFromDomains_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DomainsServiceServer is the server API for DomainsService service.
// All implementations must embed UnimplementedDomainsServiceServer
// for forward compatibility.
//
// DomainsService is a service that provides access to domains
// functionalities for magistrala services.
type DomainsServiceServer interface {
	DeleteUserFromDomains(context.Context, *DeleteUserReq) (*DeleteUserRes, error)
	mustEmbedUnimplementedDomainsServiceServer()
}

// UnimplementedDomainsServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDomainsServiceServer struct{}

func (UnimplementedDomainsServiceServer) DeleteUserFromDomains(context.Context, *DeleteUserReq) (*DeleteUserRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUserFromDomains not implemented")
}
func (UnimplementedDomainsServiceServer) mustEmbedUnimplementedDomainsServiceServer() {}
func (UnimplementedDomainsServiceServer) testEmbeddedByValue()                        {}

// UnsafeDomainsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DomainsServiceServer will
// result in compilation errors.
type UnsafeDomainsServiceServer interface {
	mustEmbedUnimplementedDomainsServiceServer()
}

func RegisterDomainsServiceServer(s grpc.ServiceRegistrar, srv DomainsServiceServer) {
	// If the following call pancis, it indicates UnimplementedDomainsServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DomainsService_ServiceDesc, srv)
}

func _DomainsService_DeleteUserFromDomains_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUserReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DomainsServiceServer).DeleteUserFromDomains(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DomainsService_DeleteUserFromDomains_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DomainsServiceServer).DeleteUserFromDomains(ctx, req.(*DeleteUserReq))
	}
	return interceptor(ctx, in, info, handler)
}

// DomainsService_ServiceDesc is the grpc.ServiceDesc for DomainsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DomainsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "domains.v1.DomainsService",
	HandlerType: (*DomainsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DeleteUserFromDomains",
			Handler:    _DomainsService_DeleteUserFromDomains_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "domains/v1/domains.proto",
}