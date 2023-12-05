// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.23.2
// source: hook.proto

package proto

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

const (
	HookFactory_GetPipelineType_FullMethodName  = "/proto.HookFactory/GetPipelineType"
	HookFactory_GetHooksMetaData_FullMethodName = "/proto.HookFactory/GetHooksMetaData"
	HookFactory_BuildHook_FullMethodName        = "/proto.HookFactory/BuildHook"
)

// HookFactoryClient is the client API for HookFactory service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type HookFactoryClient interface {
	// GetPipelineType will return the pipeline type supported by the plugin.
	GetPipelineType(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetPipelineTypeResponse, error)
	// GetHooksMetaData returns a list of metadata information for hooks supported by the plugin.
	GetHooksMetaData(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetHooksMetaDataResponse, error)
	// BuildHook will return a hook that can be executed before or after the user pipeline. It based on
	// the requested hook name, user-entered parameters, and environment information.
	BuildHook(ctx context.Context, in *BuildHookRequest, opts ...grpc.CallOption) (*BuildHookResponse, error)
}

type hookFactoryClient struct {
	cc grpc.ClientConnInterface
}

func NewHookFactoryClient(cc grpc.ClientConnInterface) HookFactoryClient {
	return &hookFactoryClient{cc}
}

func (c *hookFactoryClient) GetPipelineType(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetPipelineTypeResponse, error) {
	out := new(GetPipelineTypeResponse)
	err := c.cc.Invoke(ctx, HookFactory_GetPipelineType_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hookFactoryClient) GetHooksMetaData(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*GetHooksMetaDataResponse, error) {
	out := new(GetHooksMetaDataResponse)
	err := c.cc.Invoke(ctx, HookFactory_GetHooksMetaData_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hookFactoryClient) BuildHook(ctx context.Context, in *BuildHookRequest, opts ...grpc.CallOption) (*BuildHookResponse, error) {
	out := new(BuildHookResponse)
	err := c.cc.Invoke(ctx, HookFactory_BuildHook_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HookFactoryServer is the server API for HookFactory service.
// All implementations must embed UnimplementedHookFactoryServer
// for forward compatibility
type HookFactoryServer interface {
	// GetPipelineType will return the pipeline type supported by the plugin.
	GetPipelineType(context.Context, *Empty) (*GetPipelineTypeResponse, error)
	// GetHooksMetaData returns a list of metadata information for hooks supported by the plugin.
	GetHooksMetaData(context.Context, *Empty) (*GetHooksMetaDataResponse, error)
	// BuildHook will return a hook that can be executed before or after the user pipeline. It based on
	// the requested hook name, user-entered parameters, and environment information.
	BuildHook(context.Context, *BuildHookRequest) (*BuildHookResponse, error)
	mustEmbedUnimplementedHookFactoryServer()
}

// UnimplementedHookFactoryServer must be embedded to have forward compatible implementations.
type UnimplementedHookFactoryServer struct {
}

func (UnimplementedHookFactoryServer) GetPipelineType(context.Context, *Empty) (*GetPipelineTypeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPipelineType not implemented")
}
func (UnimplementedHookFactoryServer) GetHooksMetaData(context.Context, *Empty) (*GetHooksMetaDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHooksMetaData not implemented")
}
func (UnimplementedHookFactoryServer) BuildHook(context.Context, *BuildHookRequest) (*BuildHookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BuildHook not implemented")
}
func (UnimplementedHookFactoryServer) mustEmbedUnimplementedHookFactoryServer() {}

// UnsafeHookFactoryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to HookFactoryServer will
// result in compilation errors.
type UnsafeHookFactoryServer interface {
	mustEmbedUnimplementedHookFactoryServer()
}

func RegisterHookFactoryServer(s grpc.ServiceRegistrar, srv HookFactoryServer) {
	s.RegisterService(&HookFactory_ServiceDesc, srv)
}

func _HookFactory_GetPipelineType_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HookFactoryServer).GetPipelineType(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HookFactory_GetPipelineType_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HookFactoryServer).GetPipelineType(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _HookFactory_GetHooksMetaData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HookFactoryServer).GetHooksMetaData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HookFactory_GetHooksMetaData_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HookFactoryServer).GetHooksMetaData(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _HookFactory_BuildHook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildHookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HookFactoryServer).BuildHook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HookFactory_BuildHook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HookFactoryServer).BuildHook(ctx, req.(*BuildHookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// HookFactory_ServiceDesc is the grpc.ServiceDesc for HookFactory service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var HookFactory_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.HookFactory",
	HandlerType: (*HookFactoryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPipelineType",
			Handler:    _HookFactory_GetPipelineType_Handler,
		},
		{
			MethodName: "GetHooksMetaData",
			Handler:    _HookFactory_GetHooksMetaData_Handler,
		},
		{
			MethodName: "BuildHook",
			Handler:    _HookFactory_BuildHook_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "hook.proto",
}
