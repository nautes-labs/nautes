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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Gitlab struct {
	// Gitlab project name.
	// +kubebuilder:validation:MinLength=1
	RepoName string `json:"repoName"`
	// Supports regular expressions.
	Revision string `json:"revision"`
	// Gitlab webhook events: push_events, tag_push_events, etc.
	Events []string `json:"events"`
}

type Calendar struct {
	// Schedule is a cron-like expression. For reference, see: https://en.wikipedia.org/wiki/Cron
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// Interval is a string that describes an interval duration, e.g. 1s, 30m, 2h...
	// +optional
	Interval string `json:"interval,omitempty"`
	// ExclusionDates defines the list of DATE-TIME exceptions for recurring events.
	// +optional
	ExclusionDates []string `json:"exclusionDates,omitempty"`
	// Timezone in which to run the schedule
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

type EventSource struct {
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Gitlab *Gitlab `json:"gitlab,omitempty"`
	// +optional
	Calendar *Calendar `json:"calendar,omitempty"`
}

// The definition of event source triggered pipeline mode.
type PipelineTrigger struct {
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:MinLength=1
	EventSource string `json:"eventSource"`
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:MinLength=1
	Pipeline string `json:"pipeline"`
	// Optional
	// Regular expressions are not supported, If it is empty, the trigger will determine the revision of the pipeline based on the revision of the event source
	Revision string `json:"revision,omitempty"`
}

// The definition of a multi-branch pipeline.One pipeline corresponds to one declaration file in the Git repository.
type Pipeline struct {
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Optional
	// Default is 'default'
	Label string `json:"label,omitempty"`
	// Pipeline manifest path, wildcard support.
	Path string `json:"path"`
}

// ProjectPipelineRuntimeSpec defines the desired state of ProjectPipelineRuntime
type ProjectPipelineRuntimeSpec struct {
	Project string `json:"project"`
	// The code repo for pipeline manifests.
	PipelineSource string `json:"pipelineSource"`
	// The definition of pipeline.
	Pipelines []Pipeline `json:"pipelines"`
	// The target environment for running the pipeline.
	Destination ProjectPipelineDestination `json:"destination"`
	// Events source that may trigger the pipeline.
	EventSources []EventSource `json:"eventSources"`
	// Isolation definition of pipeline runtime related resources: shared(default) or exclusive
	Isolation        string            `json:"isolation"`
	PipelineTriggers []PipelineTrigger `json:"pipelineTriggers"`
	// optional
	AdditionalResources *ProjectPipelineRuntimeAdditionalResources `json:"additionalResources,omitempty"`
	// +optional
	// Hooks are hooks that users need to add before or after the user pipeline.
	Hooks *Hooks `json:"hooks,omitempty"`
	// +optional
	Account string `json:"account,omitempty" yaml:"account"`
}

// Hooks are hooks will to add before or after the user pipeline.
type Hooks struct {
	// +optional
	// PreHooks is a set of hooks to be executed before running the user pipeline.
	PreHooks []Hook `json:"preHooks,omitempty"`
	// +optional
	// PostHooks is a set of hooks that will run after the user pipeline starts executing.
	PostHooks []Hook `json:"postHooks,omitempty"`
}

// Hook is a record of information about a runnable hook.
type Hook struct {
	// Name is the name of the hook to be executed.
	Name string `json:"name"`
	// Alias is the alias given by the user for the hook.
	// If the user does not enter this value, the name of the hook will be obtained from 'name'.
	// When the hook appears in both PreHooks and PostHooks, it is necessary to specify the name to prevent conflicts.
	// +optional
	Alias *string `json:"alias,omitempty"`
	// Vars is the parameter that the user wants to pass to the hook,
	// and the input items are determined based on the pipeline component in cluster.
	// +optional
	Vars map[string]string `json:"vars,omitempty"`
}

// ProjectPipelineRuntimeDestination defines where pipeline runtime will run
type ProjectPipelineRuntimeDestination struct {
	Environment string `json:"environment"`
	Namespace   string `json:"namespace"`
}

// ProjectPipelineRuntimeAdditionalResources defines the additional resources witch runtime needed
type ProjectPipelineRuntimeAdditionalResources struct {
	// Optional
	Git *ProjectPipelineRuntimeAdditionalResourcesGit `json:"git,omitempty"`
}

// ProjectPipelineRuntimeAdditionalResourcesGit defines the additional resources if it comes from git
type ProjectPipelineRuntimeAdditionalResourcesGit struct {
	// Optional
	CodeRepo string `json:"codeRepo,omitempty"`
	// Optional
	// If git repo is a public repo, use url instead
	URL      string `json:"url,omitempty"`
	Revision string `json:"revision"`
	Path     string `json:"path"`
}

// ProjectPipelineRuntimeStatus defines the observed state of ProjectPipelineRuntime
type ProjectPipelineRuntimeStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	Cluster string `json:"cluster,omitempty"`
	// +optional
	// +nullable
	// IllegalEventSources records eventsources that will not be synchronized to the environment,
	// and why it will not be synchronized to the past
	IllegalEventSources []IllegalEventSource `json:"illegalEventSources"`
	// +optional
	// +nullable
	DeployStatus *runtime.RawExtension `json:"deployStatus,omitempty" yaml:"deployStatus"`
}

type IllegalEventSource struct {
	EventSource EventSource `json:"eventSource"`
	Reason      string      `json:"reason"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ppr
//+kubebuilder:printcolumn:name="Project",type=string,JSONPath=".spec.project"
//+kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.pipelinesource"

// ProjectPipelineRuntime is the Schema for the projectpipelineruntimes API
type ProjectPipelineRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectPipelineRuntimeSpec   `json:"spec,omitempty"`
	Status ProjectPipelineRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectPipelineRuntimeList contains a list of ProjectPipelineRuntime
type ProjectPipelineRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectPipelineRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectPipelineRuntime{}, &ProjectPipelineRuntimeList{})
}
