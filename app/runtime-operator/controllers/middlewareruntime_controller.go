/*
Copyright 2024.

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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// MiddlewareRuntimeReconciler reconciles a MiddlewareRuntime object
type MiddlewareRuntimeReconciler struct {
	client.Client
	Scheme *k8sruntime.Scheme
	Syncer syncer.Syncer
}

//+kubebuilder:rbac:groups=nautes.nautes.io,resources=middlewareruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.nautes.io,resources=middlewareruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.nautes.io,resources=middlewareruntimes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MiddlewareRuntime object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *MiddlewareRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	runtime := &v1alpha1.MiddlewareRuntime{}
	if err := r.Get(ctx, req.NamespacedName, runtime); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !controllerutil.ContainsFinalizer(runtime, runtimeFinalizerName) {
		controllerutil.AddFinalizer(runtime, runtimeFinalizerName)
		logger.Info("Add finalizer", "name", runtime.Name)
		if err := r.Client.Update(ctx, runtime); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update runtime: %w", err)
		}
	}

	task, err := r.Syncer.NewTask(ctx, runtime, nil)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create task: %w", err)
	}

	if !runtime.DeletionTimestamp.IsZero() {
		logger.Info("Delete runtime", "name", runtime.Name)
		if _, err := task.Delete(ctx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete task: %w", err)
		}

		controllerutil.RemoveFinalizer(runtime, runtimeFinalizerName)
		if err := r.Client.Update(ctx, runtime); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update runtime: %w", err)
		}
		return ctrl.Result{}, nil
	}

	status, err := task.Run(ctx)
	changed := setMiddlewareRuntimeStatus(logger, runtime, status, err)
	if changed {
		logger.V(1).Info("Update runtime status", "name", runtime.Name)
		if err := r.Status().Update(ctx, runtime); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update runtime status: %w", err)
		}
	}

	return ctrl.Result{RequeueAfter: reconcileFrequency}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *MiddlewareRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MiddlewareRuntime{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func setMiddlewareRuntimeStatus(logger logr.Logger, runtime *v1alpha1.MiddlewareRuntime, status *k8sruntime.RawExtension, err error) (changed bool) {
	runtimeStatus := &v1alpha1.MiddlewareRuntimeStatus{}
	if err := json.Unmarshal(status.Raw, runtimeStatus); err != nil {
		logger.Error(err, "failed to unmarshal runtime status")
		return false
	}

	lastStatus := runtime.Status.DeepCopy()
	condition := metav1.Condition{
		Type: runtimeConditionType,
	}

	if err != nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = runtimeConditionReason
		condition.Message = err.Error()
	} else {
		condition.Status = metav1.ConditionTrue
		condition.Reason = runtimeConditionReason
		condition.Message = "Success"
	}

	runtimeStatus.Conditions = v1alpha1.GetNewConditions(runtimeStatus.Conditions, []metav1.Condition{condition}, map[string]bool{runtimeConditionType: true})
	if equality.Semantic.DeepEqual(lastStatus, runtimeStatus) {
		return false
	}
	runtime.Status = *runtimeStatus

	return true
}
