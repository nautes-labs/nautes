// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.6.3
// - protoc             v3.12.4
// source: environment/v1/environment.proto

package v1

import (
	context "context"
	http "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationEnvironmentDeleteEnvironment = "/api.environment.v1.Environment/DeleteEnvironment"
const OperationEnvironmentGetEnvironment = "/api.environment.v1.Environment/GetEnvironment"
const OperationEnvironmentListEnvironments = "/api.environment.v1.Environment/ListEnvironments"
const OperationEnvironmentSaveEnvironment = "/api.environment.v1.Environment/SaveEnvironment"

type EnvironmentHTTPServer interface {
	DeleteEnvironment(context.Context, *DeleteRequest) (*DeleteReply, error)
	GetEnvironment(context.Context, *GetRequest) (*GetReply, error)
	ListEnvironments(context.Context, *ListsRequest) (*ListsReply, error)
	SaveEnvironment(context.Context, *SaveRequest) (*SaveReply, error)
}

func RegisterEnvironmentHTTPServer(s *http.Server, srv EnvironmentHTTPServer) {
	r := s.Route("/")
	r.GET("/api/v1/products/{product_name}/environments/{environmentName}", _Environment_GetEnvironment0_HTTP_Handler(srv))
	r.GET("/api/v1/products/{product_name}/environments", _Environment_ListEnvironments0_HTTP_Handler(srv))
	r.POST("/api/v1/products/{product_name}/environments/{environmentName}", _Environment_SaveEnvironment0_HTTP_Handler(srv))
	r.DELETE("/api/v1/products/{product_name}/environments/{environmentName}", _Environment_DeleteEnvironment0_HTTP_Handler(srv))
}

func _Environment_GetEnvironment0_HTTP_Handler(srv EnvironmentHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in GetRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationEnvironmentGetEnvironment)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetEnvironment(ctx, req.(*GetRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*GetReply)
		return ctx.Result(200, reply)
	}
}

func _Environment_ListEnvironments0_HTTP_Handler(srv EnvironmentHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in ListsRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationEnvironmentListEnvironments)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.ListEnvironments(ctx, req.(*ListsRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*ListsReply)
		return ctx.Result(200, reply)
	}
}

func _Environment_SaveEnvironment0_HTTP_Handler(srv EnvironmentHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in SaveRequest
		if err := ctx.Bind(&in.Body); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationEnvironmentSaveEnvironment)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.SaveEnvironment(ctx, req.(*SaveRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*SaveReply)
		return ctx.Result(200, reply)
	}
}

func _Environment_DeleteEnvironment0_HTTP_Handler(srv EnvironmentHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DeleteRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationEnvironmentDeleteEnvironment)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteEnvironment(ctx, req.(*DeleteRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*DeleteReply)
		return ctx.Result(200, reply)
	}
}

type EnvironmentHTTPClient interface {
	DeleteEnvironment(ctx context.Context, req *DeleteRequest, opts ...http.CallOption) (rsp *DeleteReply, err error)
	GetEnvironment(ctx context.Context, req *GetRequest, opts ...http.CallOption) (rsp *GetReply, err error)
	ListEnvironments(ctx context.Context, req *ListsRequest, opts ...http.CallOption) (rsp *ListsReply, err error)
	SaveEnvironment(ctx context.Context, req *SaveRequest, opts ...http.CallOption) (rsp *SaveReply, err error)
}

type EnvironmentHTTPClientImpl struct {
	cc *http.Client
}

func NewEnvironmentHTTPClient(client *http.Client) EnvironmentHTTPClient {
	return &EnvironmentHTTPClientImpl{client}
}

func (c *EnvironmentHTTPClientImpl) DeleteEnvironment(ctx context.Context, in *DeleteRequest, opts ...http.CallOption) (*DeleteReply, error) {
	var out DeleteReply
	pattern := "/api/v1/products/{product_name}/environments/{environmentName}"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationEnvironmentDeleteEnvironment))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *EnvironmentHTTPClientImpl) GetEnvironment(ctx context.Context, in *GetRequest, opts ...http.CallOption) (*GetReply, error) {
	var out GetReply
	pattern := "/api/v1/products/{product_name}/environments/{environmentName}"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationEnvironmentGetEnvironment))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *EnvironmentHTTPClientImpl) ListEnvironments(ctx context.Context, in *ListsRequest, opts ...http.CallOption) (*ListsReply, error) {
	var out ListsReply
	pattern := "/api/v1/products/{product_name}/environments"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationEnvironmentListEnvironments))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *EnvironmentHTTPClientImpl) SaveEnvironment(ctx context.Context, in *SaveRequest, opts ...http.CallOption) (*SaveReply, error) {
	var out SaveReply
	pattern := "/api/v1/products/{product_name}/environments/{environmentName}"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationEnvironmentSaveEnvironment))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in.Body, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}
