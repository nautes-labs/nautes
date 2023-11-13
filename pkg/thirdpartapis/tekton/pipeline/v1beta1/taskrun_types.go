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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// TaskRunSpec defines the desired state of TaskRun
type TaskRunSpec struct {
	Debug *TaskRunDebug `json:"debug,omitempty"`

	Params []Param `json:"params,omitempty"`

	Resources *TaskRunResources `json:"resources,omitempty"`

	ServiceAccountName string `json:"serviceAccountName"`
	// no more than one of the TaskRef and TaskSpec may be specified.

	TaskRef *TaskRef `json:"taskRef,omitempty"`

	TaskSpec *TaskSpec `json:"taskSpec,omitempty"`
	// Used for cancelling a taskrun (and maybe more later on)

	Status TaskRunSpecStatus `json:"status,omitempty"`
	// Time after which the build times out. Defaults to 1 hour.
	// Specified build timeout should be less than 24h.
	// Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration

	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// PodTemplate holds pod specific configuration
	PodTemplate *PodTemplate `json:"podTemplate,omitempty"`
	// Workspaces is a list of WorkspaceBindings from volumes to workspaces.

	Workspaces []WorkspaceBinding `json:"workspaces,omitempty"`
	// Overrides to apply to Steps in this TaskRun.
	// If a field is specified in both a Step and a StepOverride,
	// the value from the StepOverride will be used.
	// This field is only supported when the alpha feature gate is enabled.

	StepOverrides []TaskRunStepOverride `json:"stepOverrides,omitempty"`
	// Overrides to apply to Sidecars in this TaskRun.
	// If a field is specified in both a Sidecar and a SidecarOverride,
	// the value from the SidecarOverride will be used.
	// This field is only supported when the alpha feature gate is enabled.

	SidecarOverrides []TaskRunSidecarOverride `json:"sidecarOverrides,omitempty"`
}

// TaskRunSpecStatus defines the taskrun spec status the user can provide
type TaskRunSpecStatus string

const (
	// TaskRunSpecStatusCancelled indicates that the user wants to cancel the task,
	// if not already cancelled or terminated
	TaskRunSpecStatusCancelled = "TaskRunCancelled"
)

// TaskRunDebug defines the breakpoint config for a particular TaskRun
type TaskRunDebug struct {
	Breakpoint []string `json:"breakpoint,omitempty"`
}

// TaskRunInputs holds the input values that this task was invoked with.
type TaskRunInputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`

	Params []Param `json:"params,omitempty"`
}

// TaskRunOutputs holds the output values that this task was invoked with.
type TaskRunOutputs struct {
	Resources []TaskResourceBinding `json:"resources,omitempty"`
}

var taskRunCondSet = apis.NewBatchConditionSet()

// TaskRunStatus defines the observed state of TaskRun
type TaskRunStatus struct {
	duckv1beta1.Status `json:",inline"`

	// TaskRunStatusFields inlines the status fields.
	TaskRunStatusFields `json:",inline"`
}

// TaskRunReason is an enum used to store all TaskRun reason for
// the Succeeded condition that are controlled by the TaskRun itself. Failure
// reasons that emerge from underlying resources are not included here
type TaskRunReason string

const (
	// TaskRunReasonStarted is the reason set when the TaskRun has just started
	TaskRunReasonStarted TaskRunReason = "Started"
	// TaskRunReasonRunning is the reason set when the TaskRun is running
	TaskRunReasonRunning TaskRunReason = "Running"
	// TaskRunReasonSuccessful is the reason set when the TaskRun completed successfully
	TaskRunReasonSuccessful TaskRunReason = "Succeeded"
	// TaskRunReasonFailed is the reason set when the TaskRun completed with a failure
	TaskRunReasonFailed TaskRunReason = "Failed"
	// TaskRunReasonCancelled is the reason set when the Taskrun is cancelled by the user
	TaskRunReasonCancelled TaskRunReason = "TaskRunCancelled"
	// TaskRunReasonTimedOut is the reason set when the Taskrun has timed out
	TaskRunReasonTimedOut TaskRunReason = "TaskRunTimeout"
	// TaskRunReasonResolvingTaskRef indicates that the TaskRun is waiting for
	// its taskRef to be asynchronously resolved.
	TaskRunReasonResolvingTaskRef = "ResolvingTaskRef"
	// TaskRunReasonImagePullFailed is the reason set when the step of a task fails due to image not being pulled
	TaskRunReasonImagePullFailed TaskRunReason = "TaskRunImagePullFailed"
)

// TaskRunStatusFields holds the fields of TaskRun's status.  This is defined
// separately and inlined so that other types can readily consume these fields
// via duck typing.
type TaskRunStatusFields struct {
	// PodName is the name of the pod responsible for executing this task's steps.
	PodName string `json:"podName"`

	// StartTime is the time the build is actually started.

	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.

	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Steps describes the state of each build step container.

	Steps []StepState `json:"steps,omitempty"`

	// CloudEvents describe the state of each cloud event requested via a
	// CloudEventResource.

	CloudEvents []CloudEventDelivery `json:"cloudEvents,omitempty"`

	// RetriesStatus contains the history of TaskRunStatus in case of a retry in order to keep record of failures.
	// All TaskRunStatus stored in RetriesStatus will have no date within the RetriesStatus as is redundant.

	RetriesStatus []TaskRunStatus `json:"retriesStatus,omitempty"`

	// Results from Resources built during the taskRun. currently includes
	// the digest of build container images

	ResourcesResult []PipelineResourceResult `json:"resourcesResult,omitempty"`

	// TaskRunResults are the list of results written out by the task's containers

	TaskRunResults []TaskRunResult `json:"taskResults,omitempty"`

	// The list has one entry per sidecar in the manifest. Each entry is
	// represents the imageid of the corresponding sidecar.

	Sidecars []SidecarState `json:"sidecars,omitempty"`

	// TaskSpec contains the Spec from the dereferenced Task definition used to instantiate this TaskRun.
	TaskSpec *TaskSpec `json:"taskSpec,omitempty"`
}

// TaskRunStepOverride is used to override the values of a Step in the corresponding Task.
type TaskRunStepOverride struct {
	// The name of the Step to override.
	Name string `json:"name"`
	// The resource requirements to apply to the Step.
	Resources corev1.ResourceRequirements `json:"resources"`
}

// TaskRunSidecarOverride is used to override the values of a Sidecar in the corresponding Task.
type TaskRunSidecarOverride struct {
	// The name of the Sidecar to override.
	Name string `json:"name"`
	// The resource requirements to apply to the Sidecar.
	Resources corev1.ResourceRequirements `json:"resources"`
}

// StepState reports the results of running a step in a Task.
type StepState struct {
	corev1.ContainerState `json:",inline"`
	Name                  string `json:"name,omitempty"`
	ContainerName         string `json:"container,omitempty"`
	ImageID               string `json:"imageID,omitempty"`
}

// SidecarState reports the results of running a sidecar in a Task.
type SidecarState struct {
	corev1.ContainerState `json:",inline"`
	Name                  string `json:"name,omitempty"`
	ContainerName         string `json:"container,omitempty"`
	ImageID               string `json:"imageID,omitempty"`
}

// CloudEventDelivery is the target of a cloud event along with the state of
// delivery.
type CloudEventDelivery struct {
	// Target points to an addressable
	Target string                  `json:"target,omitempty"`
	Status CloudEventDeliveryState `json:"status,omitempty"`
}

// CloudEventCondition is a string that represents the condition of the event.
type CloudEventCondition string

const (
	// CloudEventConditionUnknown means that the condition for the event to be
	// triggered was not met yet, or we don't know the state yet.
	CloudEventConditionUnknown CloudEventCondition = "Unknown"
	// CloudEventConditionSent means that the event was sent successfully
	CloudEventConditionSent CloudEventCondition = "Sent"
	// CloudEventConditionFailed means that there was one or more attempts to
	// send the event, and none was successful so far.
	CloudEventConditionFailed CloudEventCondition = "Failed"
)

// CloudEventDeliveryState reports the state of a cloud event to be sent.
type CloudEventDeliveryState struct {
	// Current status
	Condition CloudEventCondition `json:"condition,omitempty"`
	// SentAt is the time at which the last attempt to send the event was made

	SentAt *metav1.Time `json:"sentAt,omitempty"`
	// Error is the text of error (if any)
	Error string `json:"message"`
	// RetryCount is the number of attempts of sending the cloud event
	RetryCount int32 `json:"retryCount"`
}

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
