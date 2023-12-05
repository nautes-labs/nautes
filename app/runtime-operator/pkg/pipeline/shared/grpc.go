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

package shared

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/proto"
	"github.com/nautes-labs/nautes/pkg/resource"
	"google.golang.org/grpc"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "NAUTES_PIPELINE_PLUGIN",
	MagicCookieValue: "hereIsNautesPipeline",
}

const (
	PluginTypeGRPC = "grpc"
)

var PluginMap = map[string]plugin.Plugin{
	PluginTypeGRPC: &HookFactoryPlugin{},
}

type HookFactoryPlugin struct {
	plugin.Plugin
	Impl component.HookFactory
}

func (hfp *HookFactoryPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterHookFactoryServer(s, &GRPCServer{Impl: hfp.Impl})
	return nil
}

func (hfp *HookFactoryPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewHookFactoryClient(c)}, nil
}

type GRPCClient struct {
	client proto.HookFactoryClient
}

func (c *GRPCClient) GetPipelineType() (string, error) {
	resp, err := c.client.GetPipelineType(context.TODO(), &proto.Empty{})
	if err != nil {
		return "", err
	}
	return resp.PipelineType, nil
}
func (c *GRPCClient) GetHooksMetadata() ([]resource.HookMetadata, error) {
	resp, err := c.client.GetHooksMetaData(context.TODO(), &proto.Empty{})
	if err != nil {
		return nil, err
	}

	metadatas := make([]resource.HookMetadata, len(resp.HooksMetaData))
	for i, metadata := range resp.HooksMetaData {
		metadatas[i] = resource.HookMetadata{
			Name:                    metadata.Name,
			IsPreHook:               metadata.IsPreHook,
			IsPostHook:              metadata.IsPostHook,
			SupportEventSourceTypes: metadata.SupportEventSourceTypes,
			VarsDefinition:          nil,
		}

		if metadata.VarsDefinition != nil {
			schema := &apiextensions.JSONSchemaProps{}
			err := json.Unmarshal(metadata.VarsDefinition, schema)
			if err != nil {
				return nil, err
			}
			metadatas[i].VarsDefinition = schema
		}
	}
	return metadatas, nil
}

func (c *GRPCClient) BuildHook(hookName string, info component.HookBuildData) (*component.Hook, error) {
	resp, err := c.client.BuildHook(context.TODO(), &proto.BuildHookRequest{
		HookName: hookName,
		Info: &proto.HookBuildData{
			UserVars:          info.UserVars,
			BuiltinVars:       info.BuiltinVars,
			EventSourceType:   info.EventSourceType,
			EventListenerType: info.EventListenerType,
		},
	})
	if err != nil {
		return nil, err
	}

	reqVars := make([]component.PluginInputOverWrite, len(resp.Hook.RequestInputs))
	for i, input := range resp.Hook.RequestInputs {
		var inputSrc component.InputSource
		src := input.Source.GetSrc()
		switch v := src.(type) {
		case *proto.InputSource_FromEventSource:
			inputSrc = component.InputSource{
				Type:            component.SourceTypeEventSource,
				FromEventSource: &v.FromEventSource,
			}
		default:
			return nil, fmt.Errorf("unknown event source type")
		}
		reqVars[i] = component.PluginInputOverWrite{
			Source:      inputSrc,
			Destination: input.Destination,
		}
	}

	var requestResources []component.RequestResource
	for i := range resp.Hook.RequestResources {
		reqRes := &component.RequestResource{}
		if err := json.Unmarshal(resp.Hook.RequestResources[i], reqRes); err != nil {
			return nil, fmt.Errorf("unmarshal resource request failed: %w", err)
		}

		requestResources = append(requestResources, *reqRes)
	}

	hook := &component.Hook{
		RequestVars:      reqVars,
		RequestResources: requestResources,
		Resource:         resp.Hook.Resource,
	}
	return hook, nil
}

type GRPCServer struct {
	Impl component.HookFactory
	*proto.UnimplementedHookFactoryServer
}

func (s *GRPCServer) GetPipelineType(_ context.Context, _ *proto.Empty) (*proto.GetPipelineTypeResponse, error) {
	pipelineType, _ := s.Impl.GetPipelineType()
	return &proto.GetPipelineTypeResponse{
		PipelineType: pipelineType,
	}, nil
}

func (s *GRPCServer) GetHooksMetaData(_ context.Context, _ *proto.Empty) (*proto.GetHooksMetaDataResponse, error) {
	metadata, _ := s.Impl.GetHooksMetadata()
	protoMetaDatas := make([]*proto.HookMetadata, len(metadata))
	for i := range metadata {
		protoMetaDatas[i] = &proto.HookMetadata{
			Name:                    metadata[i].Name,
			IsPreHook:               metadata[i].IsPreHook,
			IsPostHook:              metadata[i].IsPostHook,
			SupportEventSourceTypes: metadata[i].SupportEventSourceTypes,
			VarsDefinition:          nil,
		}

		if metadata[i].VarsDefinition != nil {
			varDef, err := json.Marshal(metadata[i].VarsDefinition)
			if err != nil {
				return nil, err
			}
			protoMetaDatas[i].VarsDefinition = varDef
		}
	}
	return &proto.GetHooksMetaDataResponse{
		HooksMetaData: protoMetaDatas,
	}, nil
}

func (s *GRPCServer) BuildHook(_ context.Context, in *proto.BuildHookRequest) (*proto.BuildHookResponse, error) {
	info := component.HookBuildData{
		UserVars:          in.Info.UserVars,
		BuiltinVars:       in.Info.BuiltinVars,
		EventSourceType:   in.Info.EventSourceType,
		EventListenerType: in.Info.EventListenerType,
	}
	hook, err := s.Impl.BuildHook(in.HookName, info)
	if err != nil {
		return nil, err
	}

	protoRequestInputs := make([]*proto.InputOverWrite, len(hook.RequestVars))
	for i := range hook.RequestVars {
		var src *proto.InputSource
		switch hook.RequestVars[i].Source.Type {
		case component.SourceTypeEventSource:
			src = &proto.InputSource{
				Src: &proto.InputSource_FromEventSource{
					FromEventSource: *hook.RequestVars[i].Source.FromEventSource,
				},
			}
		default:
			return nil, fmt.Errorf("unknown event source type")
		}
		protoRequestInputs[i] = &proto.InputOverWrite{
			Source:      src,
			Destination: hook.RequestVars[i].Destination,
		}
	}

	var requestResources [][]byte
	for i := range hook.RequestResources {
		reqByte, err := json.Marshal(hook.RequestResources[i])
		if err != nil {
			return nil, fmt.Errorf("marshal resource request failed: %w", err)
		}
		requestResources = append(requestResources, reqByte)
	}

	resp := &proto.BuildHookResponse{
		Hook: &proto.Hook{
			RequestInputs:    protoRequestInputs,
			RequestResources: requestResources,
			Resource:         hook.Resource,
		},
	}
	return resp, nil
}
