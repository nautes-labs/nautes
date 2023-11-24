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

package component

import "github.com/nautes-labs/nautes/pkg/resource"

// The PipelinePluginManager can manage all pipeline component plugins.
type PipelinePluginManager interface {
	// GetHookFactory will return a 'HookFactory' that can implement the hook based on the pipeline type and hook type.
	// If the corresponding plugin cannot be found, an error is returned.
	GetHookFactory(pipelineType, hookName string) (HookFactory, error)
}

type HookFactory interface {
	GetPipelineType() (string, error)
	GetHooksMetadata() ([]resource.HookMetadata, error)
	BuildHook(hookName string, info HookBuildData) (*Hook, error)
}

type HookBuildData struct {
	UserVars          map[string]string
	BuiltinVars       map[string]string
	EventSourceType   string
	EventListenerType string
}

type Hook struct {
	RequestVars      []PluginInputOverWrite
	RequestResources []RequestResource
	Resource         []byte
}

type PluginInputOverWrite struct {
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
