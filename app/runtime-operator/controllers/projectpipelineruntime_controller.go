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

package controllers

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ProjectPipelineRuntimeReconciler reconciles a ProjectPipelineRuntime object
type ProjectPipelineRuntimeReconciler struct {
	client.Client
	Scheme *k8sRuntime.Scheme
	Syncer syncer.Syncer
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes/finalizers,verbs=update
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepobindings,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProjectPipelineRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	runtime := &nautescrd.ProjectPipelineRuntime{}
	err := r.Client.Get(ctx, req.NamespacedName, runtime)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	task, err := r.Syncer.NewTask(ctx, runtime, runtime.Status.DeployStatus)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("create deploy task failed: %w", err)
	}

	if !runtime.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(runtime, runtimeFinalizerName) {
			return ctrl.Result{}, nil
		}

		cache, err := task.Delete(ctx)
		if err != nil {
			setPipelineRuntimeStatus(runtime, cache, err)
			if err := r.Status().Update(ctx, runtime); err != nil {
				logger.Error(err, "update status failed")
			}
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(runtime, runtimeFinalizerName)
		if err := r.Update(ctx, runtime); err != nil {
			return ctrl.Result{}, err
		}
		logger.V(1).Info("delete finish")
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(runtime, runtimeFinalizerName) {
		controllerutil.AddFinalizer(runtime, runtimeFinalizerName)
		if err := r.Update(ctx, runtime); err != nil {
			return ctrl.Result{}, err
		}
	}

	illegalEventSources, err := runtime.Validate(ctx, nautescrd.NewValidateClientFromK8s(r.Client))
	if err != nil {
		setPipelineRuntimeStatus(runtime, nil, err)
		if err := r.Status().Update(ctx, runtime); err != nil {
			logger.Error(err, "update status failed")
		}

		return ctrl.Result{}, fmt.Errorf("validate runtime failed: %w", err)
	}

	cache, err := task.Run(ctx)
	setPipelineRuntimeStatus(runtime, cache, err)

	runtime.Status.IllegalEventSources = illegalEventSources
	if err := r.Status().Update(ctx, runtime); err != nil {
		return ctrl.Result{}, err
	}
	logger.V(1).Info("reconcile finish")

	return ctrl.Result{RequeueAfter: reconcileFrequency}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectPipelineRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nautescrd.ProjectPipelineRuntime{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func NewPipelineRuntimeWithOutIllegalEventSource(runtime nautescrd.ProjectPipelineRuntime, illegalEventSources []nautescrd.IllegalEventSource) *nautescrd.ProjectPipelineRuntime {
	newRuntime := runtime.DeepCopy().DeepCopyObject().(*nautescrd.ProjectPipelineRuntime)
	// Set version to 0 to avoid syncer update resource by new runtime
	newRuntime.ResourceVersion = "0"

	if len(illegalEventSources) == 0 {
		return newRuntime
	}

	indexIllegalEventSource := map[string]bool{}
	for _, illegalEv := range illegalEventSources {
		indexIllegalEventSource[illegalEv.EventSource.Name] = true
	}

	legalEventSources := []nautescrd.EventSource{}
	for _, ev := range newRuntime.Spec.EventSources {
		if !indexIllegalEventSource[ev.Name] {
			legalEventSources = append(legalEventSources, ev)
		}
	}
	newRuntime.Spec.EventSources = legalEventSources

	legalTriggers := []nautescrd.PipelineTrigger{}
	for _, trigger := range newRuntime.Spec.PipelineTriggers {
		if !indexIllegalEventSource[trigger.EventSource] {
			legalTriggers = append(legalTriggers, trigger)
		}
	}
	newRuntime.Spec.PipelineTriggers = legalTriggers

	return newRuntime
}

func setPipelineRuntimeStatus(runtime *nautescrd.ProjectPipelineRuntime, result *k8sRuntime.RawExtension, err error) {
	if result != nil {
		runtime.Status.DeployStatus = result
	}

	if err != nil {
		condition := metav1.Condition{
			Type:    runtimeConditionType,
			Status:  "False",
			Reason:  runtimeConditionReason,
			Message: err.Error(),
		}
		runtime.Status.Conditions = nautescrd.GetNewConditions(runtime.Status.Conditions, []metav1.Condition{condition}, map[string]bool{runtimeConditionType: true})
	} else {
		condition := metav1.Condition{
			Type:   runtimeConditionType,
			Status: "True",
			Reason: runtimeConditionReason,
		}
		runtime.Status.Conditions = nautescrd.GetNewConditions(runtime.Status.Conditions, []metav1.Condition{condition}, map[string]bool{runtimeConditionType: true})
	}
}
