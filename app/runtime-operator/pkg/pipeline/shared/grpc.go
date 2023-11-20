package shared

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/proto"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

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
func (c *GRPCClient) GetHooksMetadata() ([]HookMetadata, error) {
	resp, err := c.client.GetHooksMetaData(context.TODO(), &proto.Empty{})
	if err != nil {
		return nil, err
	}

	metadatas := make([]HookMetadata, len(resp.HooksMetaData))
	for i, metadata := range resp.HooksMetaData {
		metadatas[i] = HookMetadata{
			Name:                    metadata.Name,
			IsPreHook:               metadata.IsPreHook,
			IsPostHook:              metadata.IsPostHook,
			SupportEventListeners:   metadata.SupportEventListeners,
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

func (c *GRPCClient) BuildHook(hookName string, info HookBuildData) (*Hook, error) {
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

	reqVars := make([]InputOverWrite, len(resp.Hook.RequestInputs))
	for i, input := range resp.Hook.RequestInputs {
		var inputSrc InputSource
		src := input.Source.GetSrc()
		switch v := src.(type) {
		case *proto.InputSource_FromEventSource:
			inputSrc = InputSource{
				Type:            SourceTypeEventSource,
				FromEventSource: &v.FromEventSource,
			}
		default:
			return nil, fmt.Errorf("unknown event source type")
		}
		reqVars[i] = InputOverWrite{
			Source:      inputSrc,
			Destination: input.Destination,
		}
	}

	hook := &Hook{
		RequestVars:      reqVars,
		RequestResources: resp.Hook.RequestResources,
		Resource:         resp.Hook.Resource,
	}
	return hook, nil
}

type GRPCServer struct {
	Impl HookFactory
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
			SupportEventListeners:   metadata[i].SupportEventListeners,
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
	info := HookBuildData{
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
		case SourceTypeEventSource:
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

	resp := &proto.BuildHookResponse{
		Hook: &proto.Hook{
			RequestInputs:    protoRequestInputs,
			RequestResources: hook.RequestResources,
			Resource:         hook.Resource,
		},
	}
	return resp, nil
}
