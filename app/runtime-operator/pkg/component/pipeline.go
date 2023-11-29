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

import "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

// PipelineFactory can generate pipeline-type components.
type PipelineFactory interface {
	// NewComponent will generate a new component, with three input parameters:
	// opt is the user defined component options.
	// info is the environment information used to complete the runtime deployment.(It includes the component calling method, server access method, and snapshot of nautes resources, etc.)
	// cache is the cache of the component.When the component needs to store cache, it can write the cache to cache.The runtime operator will be responsible for caching access.
	// - The data to be cached should be able to be exported in json format.
	// - The cache of the component will be updated to the remote side after tuning is complete.
	NewComponent(opt v1alpha1.Component, info *ComponentInitInfo, status interface{}) (Pipeline, error)
	// NewStatus returns a new cache object. The return value will be passed to the component through the CreateNewComponent method.
	// The input parameter is the text serialized into json by the component cache.If there is no cache, nil will be passed in.
	NewStatus(rawStatus []byte) (interface{}, error)
}

// Pipeline is a tool used by runtime operators to create pre- and post-pipeline hooks and create pipeline execution environments.
type Pipeline interface {
	Component

	// GetHooks will generate tasks to be executed before and after the pipeline based on the information in info.
	// The default steps should include pulling the pipeline file and executing the user pipeline.
	//
	// Return:
	// - The hooks to be executed and the input information required to run the hooks.
	// - A list of resources that need to be created in the space. For example, the key to access the pipeline repository.
	GetHooks(info HooksInitInfo) (*Hooks, []RequestResource, error)

	// GetPipelineDashBoardURL will return an address where user can view pipeline run information. If not, it will return null.
	GetPipelineDashBoardURL() string
}

// HooksInitInfo contains information used to create hooks.
type HooksInitInfo struct {
	// BuiltinVars stores information derived from pipeline resources.
	// For example, the URL corresponding to the code repository.For specific options, please refer to the constant definition of BuiltinVar.
	BuiltinVars map[BuiltinVar]string
	// Hooks are user-defined pre- and post-pipeline steps
	Hooks v1alpha1.Hooks
	// The EventSource stores the event sources corresponding to the pipeline.
	EventSource EventSource
	// EventSourceType is the type of event source in the event source.
	EventSourceType EventSourceType
	// EventListerType is the name of the event listener component deployed in the runtime.
	EventListenerType string
}

// HookSpace defines a space that can run specific hooks.
type HookSpace struct {
	BaseSpace Space
	// DeployResources is the resource to be deployed
	DeployResources []RequestResource
}

// Hooks contains information about all the pre- and post-tasks of the user pipeline.
type Hooks struct {
	// RequestVars defines the parameters required for running tasks.
	RequestVars []InputOverWrite
	// Resource stores the content of the specific task to be executed.
	Resource interface{}
}

// InputOverWrite defines how to pass data from the event source to the task.
type InputOverWrite struct {
	// BuiltinRequestVar is the data to be obtained.
	BuiltinRequestVar EventSourceVar
	// // StaticeVar is the event source path specified by the pipeline.
	StaticeVar *string
	// Dest is the path to be replaced in the task resource.
	// The current version only considers the addressing method of json, such as spec.data.1.value.
	Dest string
}

type ResourceType string

const (
	ResourceTypeCodeRepoSSHKey      ResourceType = "sshKey"
	ResourceTypeCodeRepoAccessToken ResourceType = "accessToken"
	ResourceTypeCAFile              ResourceType = "caFile"
	ResourceTypeEmptyDir            ResourceType = "empty"
)

type RequestResource struct {
	Type         ResourceType                `json:"type"`
	ResourceName string                      `json:"resourceName"`
	SSHKey       *ResourceRequestSSHKey      `json:"sshKey,omitempty"`
	AccessToken  *ResourceRequestAccessToken `json:"accessToken,omitempty"`
	CAFile       *ResourceRequestCAFile      `json:"caFile,omitempty"`
}

type ResourceRequestSSHKey struct {
	SecretInfo SecretInfo `json:"secretInfo"`
}

type ResourceRequestAccessToken struct {
	SecretInfo SecretInfo `json:"secretInfo"`
}

type ResourceRequestCAFile struct {
	URL string `json:"domain"`
}

// BuiltinVar is a built-in variable when generating hooks.
type BuiltinVar string

const (
	VarEventSourceCodeRepoName BuiltinVar = "EventSourceCodeRepoName" // the name of the code repo that triggered the event.
	VarEventSourceCodeRepoURL  BuiltinVar = "EventSourceCodeRepoURL"  // the url of the code repo that triggered the event.
	VarServiceAccount          BuiltinVar = "ServiceAccount"          // service account that runs hooks.
	VarNamespace               BuiltinVar = "Namespace"               // The namespace name of the running user pipeline.
	VarPipelineCodeRepoName    BuiltinVar = "PipelineCodeRepoName"    // Name of the code repo where the pipeline file is stored.
	VarPipelineCodeRepoURL     BuiltinVar = "PipelineCodeRepoURL"     // URL of the code repo where the pipeline file is stored.
	VarPipelineRevision        BuiltinVar = "PipelineRevision"        // User-specified pipeline branches.
	VarPipelineFilePath        BuiltinVar = "PipelineFilePath"        // The path of the pipeline file specified by the user.
	VarCodeRepoProviderType    BuiltinVar = "ProviderType"            // Type of code repository provider.
	VarCodeRepoProviderURL     BuiltinVar = "CodeRepoProviderURL"     // The API server URL of the code repo provider.
	VarPipelineDashBoardURL    BuiltinVar = "PipelineDashBoardURL"    // The url address to view the pipeline running status
	VarPipelineLabel           BuiltinVar = "PipelineLabel"
)

// EventSourceVar is a variable type that can be obtained from the event source.
type EventSourceVar string

const (
	EventSourceVarRef EventSourceVar = "EventSourceRef" // The ref of the code repo that triggered the event.
)
