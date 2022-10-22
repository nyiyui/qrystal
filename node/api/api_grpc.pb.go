// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.7
// source: api.proto

package api

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

// NodeClient is the client API for Node service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type NodeClient interface {
	Auth(ctx context.Context, opts ...grpc.CallOption) (Node_AuthClient, error)
	Xch(ctx context.Context, in *XchQ, opts ...grpc.CallOption) (*XchS, error)
	Ping(ctx context.Context, in *PingQS, opts ...grpc.CallOption) (*PingQS, error)
}

type nodeClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeClient(cc grpc.ClientConnInterface) NodeClient {
	return &nodeClient{cc}
}

func (c *nodeClient) Auth(ctx context.Context, opts ...grpc.CallOption) (Node_AuthClient, error) {
	stream, err := c.cc.NewStream(ctx, &Node_ServiceDesc.Streams[0], "/Node/auth", opts...)
	if err != nil {
		return nil, err
	}
	x := &nodeAuthClient{stream}
	return x, nil
}

type Node_AuthClient interface {
	Send(*AuthSQ) error
	Recv() (*AuthSQ, error)
	grpc.ClientStream
}

type nodeAuthClient struct {
	grpc.ClientStream
}

func (x *nodeAuthClient) Send(m *AuthSQ) error {
	return x.ClientStream.SendMsg(m)
}

func (x *nodeAuthClient) Recv() (*AuthSQ, error) {
	m := new(AuthSQ)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nodeClient) Xch(ctx context.Context, in *XchQ, opts ...grpc.CallOption) (*XchS, error) {
	out := new(XchS)
	err := c.cc.Invoke(ctx, "/Node/xch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) Ping(ctx context.Context, in *PingQS, opts ...grpc.CallOption) (*PingQS, error) {
	out := new(PingQS)
	err := c.cc.Invoke(ctx, "/Node/ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NodeServer is the server API for Node service.
// All implementations must embed UnimplementedNodeServer
// for forward compatibility
type NodeServer interface {
	Auth(Node_AuthServer) error
	Xch(context.Context, *XchQ) (*XchS, error)
	Ping(context.Context, *PingQS) (*PingQS, error)
	mustEmbedUnimplementedNodeServer()
}

// UnimplementedNodeServer must be embedded to have forward compatible implementations.
type UnimplementedNodeServer struct {
}

func (UnimplementedNodeServer) Auth(Node_AuthServer) error {
	return status.Errorf(codes.Unimplemented, "method Auth not implemented")
}
func (UnimplementedNodeServer) Xch(context.Context, *XchQ) (*XchS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Xch not implemented")
}
func (UnimplementedNodeServer) Ping(context.Context, *PingQS) (*PingQS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedNodeServer) mustEmbedUnimplementedNodeServer() {}

// UnsafeNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NodeServer will
// result in compilation errors.
type UnsafeNodeServer interface {
	mustEmbedUnimplementedNodeServer()
}

func RegisterNodeServer(s grpc.ServiceRegistrar, srv NodeServer) {
	s.RegisterService(&Node_ServiceDesc, srv)
}

func _Node_Auth_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NodeServer).Auth(&nodeAuthServer{stream})
}

type Node_AuthServer interface {
	Send(*AuthSQ) error
	Recv() (*AuthSQ, error)
	grpc.ServerStream
}

type nodeAuthServer struct {
	grpc.ServerStream
}

func (x *nodeAuthServer) Send(m *AuthSQ) error {
	return x.ServerStream.SendMsg(m)
}

func (x *nodeAuthServer) Recv() (*AuthSQ, error) {
	m := new(AuthSQ)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Node_Xch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(XchQ)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).Xch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Node/xch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).Xch(ctx, req.(*XchQ))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingQS)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Node/ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).Ping(ctx, req.(*PingQS))
	}
	return interceptor(ctx, in, info, handler)
}

// Node_ServiceDesc is the grpc.ServiceDesc for Node service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Node_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Node",
	HandlerType: (*NodeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "xch",
			Handler:    _Node_Xch_Handler,
		},
		{
			MethodName: "ping",
			Handler:    _Node_Ping_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "auth",
			Handler:       _Node_Auth_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "api.proto",
}

// CentralSourceClient is the client API for CentralSource service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CentralSourceClient interface {
	Pull(ctx context.Context, in *PullQ, opts ...grpc.CallOption) (CentralSource_PullClient, error)
	Push(ctx context.Context, in *PushQ, opts ...grpc.CallOption) (*PushS, error)
	AddToken(ctx context.Context, in *AddTokenQ, opts ...grpc.CallOption) (*AddTokenS, error)
	CanForward(ctx context.Context, in *CanForwardQ, opts ...grpc.CallOption) (*CanForwardS, error)
	Ping(ctx context.Context, in *PingQS, opts ...grpc.CallOption) (*PingQS, error)
}

type centralSourceClient struct {
	cc grpc.ClientConnInterface
}

func NewCentralSourceClient(cc grpc.ClientConnInterface) CentralSourceClient {
	return &centralSourceClient{cc}
}

func (c *centralSourceClient) Pull(ctx context.Context, in *PullQ, opts ...grpc.CallOption) (CentralSource_PullClient, error) {
	stream, err := c.cc.NewStream(ctx, &CentralSource_ServiceDesc.Streams[0], "/CentralSource/pull", opts...)
	if err != nil {
		return nil, err
	}
	x := &centralSourcePullClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type CentralSource_PullClient interface {
	Recv() (*PullS, error)
	grpc.ClientStream
}

type centralSourcePullClient struct {
	grpc.ClientStream
}

func (x *centralSourcePullClient) Recv() (*PullS, error) {
	m := new(PullS)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *centralSourceClient) Push(ctx context.Context, in *PushQ, opts ...grpc.CallOption) (*PushS, error) {
	out := new(PushS)
	err := c.cc.Invoke(ctx, "/CentralSource/push", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *centralSourceClient) AddToken(ctx context.Context, in *AddTokenQ, opts ...grpc.CallOption) (*AddTokenS, error) {
	out := new(AddTokenS)
	err := c.cc.Invoke(ctx, "/CentralSource/addToken", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *centralSourceClient) CanForward(ctx context.Context, in *CanForwardQ, opts ...grpc.CallOption) (*CanForwardS, error) {
	out := new(CanForwardS)
	err := c.cc.Invoke(ctx, "/CentralSource/canForward", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *centralSourceClient) Ping(ctx context.Context, in *PingQS, opts ...grpc.CallOption) (*PingQS, error) {
	out := new(PingQS)
	err := c.cc.Invoke(ctx, "/CentralSource/ping", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CentralSourceServer is the server API for CentralSource service.
// All implementations must embed UnimplementedCentralSourceServer
// for forward compatibility
type CentralSourceServer interface {
	Pull(*PullQ, CentralSource_PullServer) error
	Push(context.Context, *PushQ) (*PushS, error)
	AddToken(context.Context, *AddTokenQ) (*AddTokenS, error)
	CanForward(context.Context, *CanForwardQ) (*CanForwardS, error)
	Ping(context.Context, *PingQS) (*PingQS, error)
	mustEmbedUnimplementedCentralSourceServer()
}

// UnimplementedCentralSourceServer must be embedded to have forward compatible implementations.
type UnimplementedCentralSourceServer struct {
}

func (UnimplementedCentralSourceServer) Pull(*PullQ, CentralSource_PullServer) error {
	return status.Errorf(codes.Unimplemented, "method Pull not implemented")
}
func (UnimplementedCentralSourceServer) Push(context.Context, *PushQ) (*PushS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Push not implemented")
}
func (UnimplementedCentralSourceServer) AddToken(context.Context, *AddTokenQ) (*AddTokenS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddToken not implemented")
}
func (UnimplementedCentralSourceServer) CanForward(context.Context, *CanForwardQ) (*CanForwardS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CanForward not implemented")
}
func (UnimplementedCentralSourceServer) Ping(context.Context, *PingQS) (*PingQS, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedCentralSourceServer) mustEmbedUnimplementedCentralSourceServer() {}

// UnsafeCentralSourceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CentralSourceServer will
// result in compilation errors.
type UnsafeCentralSourceServer interface {
	mustEmbedUnimplementedCentralSourceServer()
}

func RegisterCentralSourceServer(s grpc.ServiceRegistrar, srv CentralSourceServer) {
	s.RegisterService(&CentralSource_ServiceDesc, srv)
}

func _CentralSource_Pull_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(PullQ)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(CentralSourceServer).Pull(m, &centralSourcePullServer{stream})
}

type CentralSource_PullServer interface {
	Send(*PullS) error
	grpc.ServerStream
}

type centralSourcePullServer struct {
	grpc.ServerStream
}

func (x *centralSourcePullServer) Send(m *PullS) error {
	return x.ServerStream.SendMsg(m)
}

func _CentralSource_Push_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushQ)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CentralSourceServer).Push(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/CentralSource/push",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CentralSourceServer).Push(ctx, req.(*PushQ))
	}
	return interceptor(ctx, in, info, handler)
}

func _CentralSource_AddToken_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddTokenQ)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CentralSourceServer).AddToken(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/CentralSource/addToken",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CentralSourceServer).AddToken(ctx, req.(*AddTokenQ))
	}
	return interceptor(ctx, in, info, handler)
}

func _CentralSource_CanForward_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CanForwardQ)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CentralSourceServer).CanForward(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/CentralSource/canForward",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CentralSourceServer).CanForward(ctx, req.(*CanForwardQ))
	}
	return interceptor(ctx, in, info, handler)
}

func _CentralSource_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingQS)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CentralSourceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/CentralSource/ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CentralSourceServer).Ping(ctx, req.(*PingQS))
	}
	return interceptor(ctx, in, info, handler)
}

// CentralSource_ServiceDesc is the grpc.ServiceDesc for CentralSource service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CentralSource_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "CentralSource",
	HandlerType: (*CentralSourceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "push",
			Handler:    _CentralSource_Push_Handler,
		},
		{
			MethodName: "addToken",
			Handler:    _CentralSource_AddToken_Handler,
		},
		{
			MethodName: "canForward",
			Handler:    _CentralSource_CanForward_Handler,
		},
		{
			MethodName: "ping",
			Handler:    _CentralSource_Ping_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "pull",
			Handler:       _CentralSource_Pull_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api.proto",
}
