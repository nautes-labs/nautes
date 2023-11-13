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
	"k8s.io/apimachinery/pkg/runtime"

	runv1alpha1 "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/run/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

// PipelineRun represents a single execution of a Pipeline. PipelineRuns are how
// the graph of Tasks declared in a Pipeline are executed; they specify inputs
// to Pipelines such as parameter values and capture operational aspects of the
// Tasks execution such as service account and tolerations. Creating a
// PipelineRun creates TaskRuns for Tasks in the referenced Pipeline.
//

type PipelineRun struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineRunSpec `json:"spec,omitempty"`

	Status PipelineRunStatus `json:"status,omitempty"`
}

// PipelineRunSpec defines the desired state of PipelineRun
type PipelineRunSpec struct {
	PipelineRef *PipelineRef `json:"pipelineRef,omitempty"`

	PipelineSpec *PipelineSpec `json:"pipelineSpec,omitempty"`
	// Resources is a list of bindings specifying which actual instances of
	// PipelineResources to use for the resources the Pipeline has declared
	// it needs.

	Resources []PipelineResourceBinding `json:"resources,omitempty"`
	// Params is a list of parameter names and values.

	Params []Param `json:"params,omitempty"`

	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Used for cancelling a pipelinerun (and maybe more later on)

	Status PipelineRunSpecStatus `json:"status,omitempty"`
	// Time after which the Pipeline times out.
	// Currently three keys are accepted in the map
	// pipeline, tasks and finally
	// with Timeouts.pipeline >= Timeouts.tasks + Timeouts.finally

	Timeouts *TimeoutFields `json:"timeouts,omitempty"`

	// Timeout Deprecated: use pipelineRunSpec.Timeouts.Pipeline instead
	// Time after which the Pipeline times out. Defaults to never.
	// Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration

	Timeout *metav1.Duration `json:"timeout,omitempty"`
	// PodTemplate holds pod specific configuration
	PodTemplate *PodTemplate `json:"podTemplate,omitempty"`
	// Workspaces holds a set of workspace bindings that must match names
	// with those declared in the pipeline.

	Workspaces []WorkspaceBinding `json:"workspaces,omitempty"`
	// TaskRunSpecs holds a set of runtime specs

	TaskRunSpecs []PipelineTaskRunSpec `json:"taskRunSpecs,omitempty"`
}

// TimeoutFields allows granular specification of pipeline, task, and finally timeouts
type TimeoutFields struct {
	// Pipeline sets the maximum allowed duration for execution of the entire pipeline. The sum of individual timeouts for tasks and finally must not exceed this value.
	Pipeline *metav1.Duration `json:"pipeline,omitempty"`
	// Tasks sets the maximum allowed duration of this pipeline's tasks
	Tasks *metav1.Duration `json:"tasks,omitempty"`
	// Finally sets the maximum allowed duration of this pipeline's finally
	Finally *metav1.Duration `json:"finally,omitempty"`
}

// PipelineRunSpecStatus defines the pipelinerun spec status the user can provide
type PipelineRunSpecStatus string

const (
	// PipelineRunSpecStatusCancelledDeprecated Deprecated: indicates that the user wants to cancel the task,
	// if not already cancelled or terminated (replaced by "Cancelled")
	PipelineRunSpecStatusCancelledDeprecated = "PipelineRunCancelled"

	// PipelineRunSpecStatusCancelled indicates that the user wants to cancel the task,
	// if not already cancelled or terminated
	PipelineRunSpecStatusCancelled = "Cancelled"

	// PipelineRunSpecStatusCancelledRunFinally indicates that the user wants to cancel the pipeline run,
	// if not already cancelled or terminated, but ensure finally is run normally
	PipelineRunSpecStatusCancelledRunFinally = "CancelledRunFinally"

	// PipelineRunSpecStatusStoppedRunFinally indicates that the user wants to stop the pipeline run,
	// wait for already running tasks to be completed and run finally
	// if not already cancelled or terminated
	PipelineRunSpecStatusStoppedRunFinally = "StoppedRunFinally"

	// PipelineRunSpecStatusPending indicates that the user wants to postpone starting a PipelineRun
	// until some condition is met
	PipelineRunSpecStatusPending = "PipelineRunPending"
)

// PipelineRef can be used to refer to a specific instance of a Pipeline.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type PipelineRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name,omitempty"`
	// API version of the referent

	APIVersion string `json:"apiVersion,omitempty"`
	// Bundle url reference to a Tekton Bundle.

	Bundle string `json:"bundle,omitempty"`

	// ResolverRef allows referencing a Pipeline in a remote location
	// like a git repo. This field is only supported when the alpha
	// feature gate is enabled.

	ResolverRef `json:",omitempty"`
}

// PipelineRunStatus defines the observed state of PipelineRun
type PipelineRunStatus struct {
	duckv1beta1.Status `json:",inline"`

	// PipelineRunStatusFields inlines the status fields.
	PipelineRunStatusFields `json:",inline"`
}

// PipelineRunReason represents a reason for the pipeline run "Succeeded" condition
type PipelineRunReason string

const (
	// PipelineRunReasonStarted is the reason set when the PipelineRun has just started
	PipelineRunReasonStarted PipelineRunReason = "Started"
	// PipelineRunReasonRunning is the reason set when the PipelineRun is running
	PipelineRunReasonRunning PipelineRunReason = "Running"
	// PipelineRunReasonSuccessful is the reason set when the PipelineRun completed successfully
	PipelineRunReasonSuccessful PipelineRunReason = "Succeeded"
	// PipelineRunReasonCompleted is the reason set when the PipelineRun completed successfully with one or more skipped Tasks
	PipelineRunReasonCompleted PipelineRunReason = "Completed"
	// PipelineRunReasonFailed is the reason set when the PipelineRun completed with a failure
	PipelineRunReasonFailed PipelineRunReason = "Failed"
	// PipelineRunReasonCancelled is the reason set when the PipelineRun cancelled by the user
	// This reason may be found with a corev1.ConditionFalse status, if the cancellation was processed successfully
	// This reason may be found with a corev1.ConditionUnknown status, if the cancellation is being processed or failed
	PipelineRunReasonCancelled PipelineRunReason = "Cancelled"
	// PipelineRunReasonPending is the reason set when the PipelineRun is in the pending state
	PipelineRunReasonPending PipelineRunReason = "PipelineRunPending"
	// PipelineRunReasonTimedOut is the reason set when the PipelineRun has timed out
	PipelineRunReasonTimedOut PipelineRunReason = "PipelineRunTimeout"
	// PipelineRunReasonStopping indicates that no new Tasks will be scheduled by the controller, and the
	// pipeline will stop once all running tasks complete their work
	PipelineRunReasonStopping PipelineRunReason = "PipelineRunStopping"
	// PipelineRunReasonCancelledRunningFinally indicates that pipeline has been gracefully cancelled
	// and no new Tasks will be scheduled by the controller, but final tasks are now running
	PipelineRunReasonCancelledRunningFinally PipelineRunReason = "CancelledRunningFinally"
	// PipelineRunReasonStoppedRunningFinally indicates that pipeline has been gracefully stopped
	// and no new Tasks will be scheduled by the controller, but final tasks are now running
	PipelineRunReasonStoppedRunningFinally PipelineRunReason = "StoppedRunningFinally"
)

var pipelineRunCondSet = apis.NewBatchConditionSet()

// ChildStatusReference is used to point to the statuses of individual TaskRuns and Runs within this PipelineRun.
type ChildStatusReference struct {
	runtime.TypeMeta `json:",inline"`
	// Name is the name of the TaskRun or Run this is referencing.
	Name string `json:"name,omitempty"`
	// PipelineTaskName is the name of the PipelineTask this is referencing.
	PipelineTaskName string `json:"pipelineTaskName,omitempty"`

	// WhenExpressions is the list of checks guarding the execution of the PipelineTask

	WhenExpressions []WhenExpression `json:"whenExpressions,omitempty"`
}

// PipelineRunStatusFields holds the fields of PipelineRunStatus' status.
// This is defined separately and inlined so that other types can readily
// consume these fields via duck typing.
type PipelineRunStatusFields struct {
	// StartTime is the time the PipelineRun is actually started.

	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the PipelineRun completed.

	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Deprecated - use ChildReferences instead.
	// map of PipelineRunTaskRunStatus with the taskRun name as the key

	TaskRuns map[string]*PipelineRunTaskRunStatus `json:"taskRuns,omitempty"`

	// Deprecated - use ChildReferences instead.
	// map of PipelineRunRunStatus with the run name as the key

	Runs map[string]*PipelineRunRunStatus `json:"runs,omitempty"`

	// PipelineResults are the list of results written out by the pipeline task's containers

	PipelineResults []PipelineRunResult `json:"pipelineResults,omitempty"`

	// PipelineRunSpec contains the exact spec used to instantiate the run
	PipelineSpec *PipelineSpec `json:"pipelineSpec,omitempty"`

	// list of tasks that were skipped due to when expressions evaluating to false

	SkippedTasks []SkippedTask `json:"skippedTasks,omitempty"`

	// list of TaskRun and Run names, PipelineTask names, and API versions/kinds for children of this PipelineRun.

	ChildReferences []ChildStatusReference `json:"childReferences,omitempty"`
}

// SkippedTask is used to describe the Tasks that were skipped due to their When Expressions
// evaluating to False. This is a struct because we are looking into including more details
// about the When Expressions that caused this Task to be skipped.
type SkippedTask struct {
	// Name is the Pipeline Task name
	Name string `json:"name"`
	// Reason is the cause of the PipelineTask being skipped.
	Reason SkippingReason `json:"reason"`
	// WhenExpressions is the list of checks guarding the execution of the PipelineTask

	WhenExpressions []WhenExpression `json:"whenExpressions,omitempty"`
}

// SkippingReason explains why a PipelineTask was skipped.
type SkippingReason string

const (
	// WhenExpressionsSkip means the task was skipped due to at least one of its when expressions evaluating to false
	WhenExpressionsSkip SkippingReason = "When Expressions evaluated to false"
	// ParentTasksSkip means the task was skipped because its parent was skipped
	ParentTasksSkip SkippingReason = "Parent Tasks were skipped"
	// StoppingSkip means the task was skipped because the pipeline run is stopping
	StoppingSkip SkippingReason = "PipelineRun was stopping"
	// GracefullyCancelledSkip means the task was skipped because the pipeline run has been gracefully cancelled
	GracefullyCancelledSkip SkippingReason = "PipelineRun was gracefully cancelled"
	// GracefullyStoppedSkip means the task was skipped because the pipeline run has been gracefully stopped
	GracefullyStoppedSkip SkippingReason = "PipelineRun was gracefully stopped"
	// MissingResultsSkip means the task was skipped because it's missing necessary results
	MissingResultsSkip SkippingReason = "Results were missing"
	// None means the task was not skipped
	None SkippingReason = "None"
)

// PipelineRunResult used to describe the results of a pipeline
type PipelineRunResult struct {
	// Name is the result's name as declared by the Pipeline
	Name string `json:"name"`

	// Value is the result returned from the execution of this PipelineRun
	Value string `json:"value"`
}

// PipelineRunTaskRunStatus contains the name of the PipelineTask for this TaskRun and the TaskRun's Status
type PipelineRunTaskRunStatus struct {
	// PipelineTaskName is the name of the PipelineTask.
	PipelineTaskName string `json:"pipelineTaskName,omitempty"`
	// Status is the TaskRunStatus for the corresponding TaskRun

	Status *TaskRunStatus `json:"status,omitempty"`
	// WhenExpressions is the list of checks guarding the execution of the PipelineTask

	WhenExpressions []WhenExpression `json:"whenExpressions,omitempty"`
}

// PipelineRunRunStatus contains the name of the PipelineTask for this Run and the Run's Status
type PipelineRunRunStatus struct {
	// PipelineTaskName is the name of the PipelineTask.
	PipelineTaskName string `json:"pipelineTaskName,omitempty"`
	// Status is the RunStatus for the corresponding Run

	Status *runv1alpha1.RunStatus `json:"status,omitempty"`
	// WhenExpressions is the list of checks guarding the execution of the PipelineTask

	WhenExpressions []WhenExpression `json:"whenExpressions,omitempty"`
}

// PipelineRunList contains a list of PipelineRun
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items,omitempty"`
}

// PipelineTaskRun reports the results of running a step in the Task. Each
// task has the potential to succeed or fail (based on the exit code)
// and produces logs.
type PipelineTaskRun struct {
	Name string `json:"name,omitempty"`
}

// PipelineTaskRunSpec  can be used to configure specific
// specs for a concrete Task
type PipelineTaskRunSpec struct {
	PipelineTaskName       string       `json:"pipelineTaskName,omitempty"`
	TaskServiceAccountName string       `json:"taskServiceAccountName,omitempty"`
	TaskPodTemplate        *PodTemplate `json:"taskPodTemplate,omitempty"`

	StepOverrides []TaskRunStepOverride `json:"stepOverrides,omitempty"`

	SidecarOverrides []TaskRunSidecarOverride `json:"sidecarOverrides,omitempty"`

	Metadata *PipelineTaskMetadata `json:"metadata,omitempty"`
}
