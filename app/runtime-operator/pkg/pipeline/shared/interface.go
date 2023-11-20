package shared

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/proto"
	"google.golang.org/grpc"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "NAUTES_PIPELINE_PLUGIN",
	MagicCookieValue: "hereIsNautesPipeline",
}

const (
	PluginTypeGRPC = "grpc"
)

const ConfigMapNameHooksMetadata = "nautes-hooks-metadata"

var PluginMap = map[string]plugin.Plugin{
	PluginTypeGRPC: &HookFactoryPlugin{},
}

type HookFactory interface {
	GetPipelineType() (string, error)
	GetHooksMetadata() ([]HookMetadata, error)
	BuildHook(hookName string, info HookBuildData) (*Hook, error)
}

type HookMetadata struct {
	Name                    string                         `json:"name"`
	IsPreHook               bool                           `json:"isPreHook"`
	IsPostHook              bool                           `json:"isPostHook"`
	SupportEventListeners   []string                       `json:"supportEventListeners,omitempty"`
	SupportEventSourceTypes []string                       `json:"supportEventSourceTypes,omitempty"`
	VarsDefinition          *apiextensions.JSONSchemaProps `json:"varsDefinition,omitempty"`
}

type HookFactoryPlugin struct {
	plugin.Plugin
	Impl HookFactory
}

func (hfp *HookFactoryPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterHookFactoryServer(s, &GRPCServer{Impl: hfp.Impl})
	return nil
}

func (hfp *HookFactoryPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewHookFactoryClient(c)}, nil
}

type HookBuildData struct {
	UserVars          map[string]string
	BuiltinVars       map[string]string
	EventSourceType   string
	EventListenerType string
}

type Hook struct {
	RequestVars      []InputOverWrite
	RequestResources [][]byte
	Resource         []byte
}

type InputOverWrite struct {
	Source      InputSource
	Destination string
}

type SourceType string

const (
	SourceTypeEventSource SourceType = "FromEventSource"
)

type InputSource struct {
	Type            SourceType
	FromEventSource *string
}
