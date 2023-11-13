/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1beta1"
	runv1alpha1 "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/run/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
)

// EmbeddedRunSpec allows custom task definitions to be embedded
type EmbeddedRunSpec struct {
	runtime.TypeMeta `json:",inline"`

	Metadata v1beta1.PipelineTaskMetadata `json:"metadata,omitempty"`

	// Spec is a specification of a custom task

	Spec runtime.RawExtension `json:"spec,omitempty"`
}

// RunSpec defines the desired state of Run
type RunSpec struct {
	Ref *TaskRef `json:"ref,omitempty"`

	// Spec is a specification of a custom task

	Spec *EmbeddedRunSpec `json:"spec,omitempty"`

	Params []v1beta1.Param `json:"params,omitempty"`

	// Used for cancelling a run (and maybe more later on)

	Status RunSpecStatus `json:"status,omitempty"`

	// Used for propagating retries count to custom tasks

	Retries int `json:"retries,omitempty"`

	ServiceAccountName string `json:"serviceAccountName"`

	// PodTemplate holds pod specific configuration

	PodTemplate *PodTemplate `json:"podTemplate,omitempty"`

	// Time after which the custom-task times out.
	// Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration

	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Workspaces is a list of WorkspaceBindings from volumes to workspaces.

	Workspaces []v1beta1.WorkspaceBinding `json:"workspaces,omitempty"`
}

// RunSpecStatus defines the taskrun spec status the user can provide
type RunSpecStatus string

const (
	// RunSpecStatusCancelled indicates that the user wants to cancel the run,
	// if not already cancelled or terminated
	RunSpecStatusCancelled RunSpecStatus = "RunCancelled"
)

const (
	// RunReasonCancelled must be used in the Condition Reason to indicate that a Run was cancelled.
	RunReasonCancelled = "RunCancelled"
	// RunReasonTimedOut must be used in the Condition Reason to indicate that a Run was timed out.
	RunReasonTimedOut = "RunTimedOut"
	// RunReasonWorkspaceNotSupported can be used in the Condition Reason to indicate that the
	// Run contains a workspace which is not supported by this custom task.
	RunReasonWorkspaceNotSupported = "RunWorkspaceNotSupported"
	// RunReasonPodTemplateNotSupported can be used in the Condition Reason to indicate that the
	// Run contains a pod template which is not supported by this custom task.
	RunReasonPodTemplateNotSupported = "RunPodTemplateNotSupported"
)

// RunStatus defines the observed state of Run.
type RunStatus = runv1alpha1.RunStatus

var runCondSet = apis.NewBatchConditionSet()

// RunStatusFields holds the fields of Run's status.  This is defined
// separately and inlined so that other types can readily consume these fields
// via duck typing.
type RunStatusFields = runv1alpha1.RunStatusFields

// RunResult used to describe the results of a task
type RunResult = runv1alpha1.RunResult

// Run represents a single execution of a Custom Task.
//

type Run struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RunSpec `json:"spec,omitempty"`

	Status RunStatus `json:"status,omitempty"`
}

// RunList contains a list of Run
type RunList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Run `json:"items"`
}
