/*
Copyright 2019 The Tekton Authors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskRunSpec defines the desired state of TaskRun
type TaskRunSpec struct {
	ServiceAccountName string `json:"serviceAccountName"`
	// no more than one of the TaskRef and TaskSpec may be specified.

	TaskRef *TaskRef `json:"taskRef,omitempty"`

	TaskSpec *TaskSpec `json:"taskSpec,omitempty"`
	// Used for cancelling a taskrun (and maybe more later on)

	Status TaskRunSpecStatus `json:"status,omitempty"`
	// Time after which the build times out. Defaults to 10 minutes.
	// Specified build timeout should be less than 24h.
	// Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration

	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// PodTemplate holds pod specific configuration

	PodTemplate *PodTemplate `json:"podTemplate,omitempty"`
	// Workspaces is a list of WorkspaceBindings from volumes to workspaces.

	Workspaces []WorkspaceBinding `json:"workspaces,omitempty"`
	// From v1beta1

	Params []Param `json:"params,omitempty"`

	Resources *v1beta1.TaskRunResources `json:"resources,omitempty"`
	// Deprecated

	Inputs *TaskRunInputs `json:"inputs,omitempty"`

	Outputs *TaskRunOutputs `json:"outputs,omitempty"`
}

// TaskRunSpecStatus defines the taskrun spec status the user can provide
type TaskRunSpecStatus = v1beta1.TaskRunSpecStatus

const (
	// TaskRunSpecStatusCancelled indicates that the user wants to cancel the task,
	// if not already cancelled or terminated
	TaskRunSpecStatusCancelled = v1beta1.TaskRunSpecStatusCancelled

	// TaskRunReasonCancelled indicates that the TaskRun has been cancelled
	// because it was requested so by the user
	TaskRunReasonCancelled = v1beta1.TaskRunSpecStatusCancelled
)

// TaskRunInputs holds the input values that this task was invoked with.
type TaskRunInputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`

	Params []Param `json:"params,omitempty"`
}

// TaskResourceBinding points to the PipelineResource that
// will be used for the Task input or output called Name.
type TaskResourceBinding = v1beta1.TaskResourceBinding

// TaskRunOutputs holds the output values that this task was invoked with.
type TaskRunOutputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`
}

// TaskRunStatus defines the observed state of TaskRun
type TaskRunStatus = v1beta1.TaskRunStatus

// TaskRunStatusFields holds the fields of TaskRun's status.  This is defined
// separately and inlined so that other types can readily consume these fields
// via duck typing.
type TaskRunStatusFields = v1beta1.TaskRunStatusFields

// TaskRunResult used to describe the results of a task
type TaskRunResult = v1beta1.TaskRunResult

// StepState reports the results of running a step in the Task.
type StepState = v1beta1.StepState

// SidecarState reports the results of sidecar in the Task.
type SidecarState = v1beta1.SidecarState

// CloudEventDelivery is the target of a cloud event along with the state of
// delivery.
type CloudEventDelivery = v1beta1.CloudEventDelivery

// CloudEventCondition is a string that represents the condition of the event.
type CloudEventCondition = v1beta1.CloudEventCondition

const (
	// CloudEventConditionUnknown means that the condition for the event to be
	// triggered was not met yet, or we don't know the state yet.
	CloudEventConditionUnknown CloudEventCondition = v1beta1.CloudEventConditionUnknown
	// CloudEventConditionSent means that the event was sent successfully
	CloudEventConditionSent CloudEventCondition = v1beta1.CloudEventConditionSent
	// CloudEventConditionFailed means that there was one or more attempts to
	// send the event, and none was successful so far.
	CloudEventConditionFailed CloudEventCondition = v1beta1.CloudEventConditionFailed
)

// CloudEventDeliveryState reports the state of a cloud event to be sent.
type CloudEventDeliveryState = v1beta1.CloudEventDeliveryState

// TaskRun represents a single execution of a Task. TaskRuns are how the steps
// specified in a Task are executed; they specify the parameters and resources
// used to run the steps in a Task.
//

type TaskRun struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TaskRunSpec `json:"spec,omitempty"`

	Status TaskRunStatus `json:"status,omitempty"`
}

// TaskRunList contains a list of TaskRun
type TaskRunList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskRun `json:"items"`
}
