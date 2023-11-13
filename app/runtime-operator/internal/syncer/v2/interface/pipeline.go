package component

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

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
	GetHooks(info HooksInitInfo) (Hooks, []interface{}, error)

	// CreateHookSpace will transform the space where the runtime is located into a space that can run hooks.
	// It will deploy the resources required for running hooks in the 'BaseSpace' based on the 'DeployResources'.
	// The authInfo is the credential to obtain key information from secret management.
	// When the runtime has already deployed the hook space, delete invalid resources and deploy new resources based on the new 'DeployResources'.
	// Return an error when "BaseSpace" does not exist.
	CreateHookSpace(ctx context.Context, authInfo AuthInfo, space HookSpace) error

	// CleanUpHookSpace will clean up the resources on the runtime's hook space.
	// If runtime does not have hook space, return nil.
	CleanUpHookSpace(ctx context.Context) error
}

// HooksInitInfo contains information used to create hooks.
type HooksInitInfo struct {
	// BuildInVars stores information derived from pipeline resources.
	// For example, the URL corresponding to the code repository.For specific options, please refer to the constant definition of BuildInVar.
	BuildInVars map[BuildInVar]string
	// Hooks are user-defined pre- and post-pipeline steps
	Hooks v1alpha1.Hook
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
	DeployResources []interface{}
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
	// Name is the data to be obtained.
	Name EventSourceVar
	// Dest is the path to be replaced in the task resource.
	// The current version only considers the addressing method of json, such as spec.data.1.value.
	Dest string
}

// BuildInVar is a built-in variable when generating hooks.
type BuildInVar string

const (
	VarEventSourceCodeRepoName BuildInVar = "EventSourceCodeRepoName" // the name of the code repo that triggered the event.
	VarEventSourceCodeRepoURL  BuildInVar = "EventSourceCodeRepoURL"  // the url of the code repo that triggered the event.
	VarServiceAccount          BuildInVar = "ServiceAccount"          // service account that runs hooks.
	VarNamespace               BuildInVar = "Namespace"               // The namespace name of the running user pipeline.
	VarPipelineCodeRepoName    BuildInVar = "PipelineCodeRepoName"    // Name of the code repo where the pipeline file is stored.
	VarPipelineCodeRepoURL     BuildInVar = "PipelineCodeRepoURL"     // URL of the code repo where the pipeline file is stored.
	VarPipelineRevision        BuildInVar = "PipelineRevision"        // User-specified pipeline branches.
	VarPipelineFilePath        BuildInVar = "PipelineFilePath"        // The path of the pipeline file specified by the user.
	VarCodeRepoProviderType    BuildInVar = "ProviderType"            // Type of code repository provider.
	VarPipelineLabel           BuildInVar = "PipelineLabel"
)

// EventSourceVar is a variable type that can be obtained from the event source.
type EventSourceVar string

const (
	EventSourceVarRef EventSourceVar = "EventSourceRef" // The ref of the code repo that triggered the event.
)
