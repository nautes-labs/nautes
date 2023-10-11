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
	"time"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	// Required for Watching
)

const (
	runtimeFinalizerName   = "runtime.nautes.resource.nautes.io/finalizers"
	runtimeConditionType   = "RuntimeDeployed"
	runtimeConditionReason = "RegularUpdate"
)

const (
	codeRepoMapField = ".spec.manifestSource.codeRepo"
)

// DeploymentRuntimeReconciler reconciles a DeploymentRuntime object
type DeploymentRuntimeReconciler struct {
	client.Client
	Scheme *k8sRuntime.Scheme
	Syncer syncer.Syncer
}

var reconcileFrequency = time.Second * 60
var errorMsgUpdateStatusFailed = "update status failed"

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes/finalizers,verbs=update
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoproviders,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=artifactrepoes,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=artifactrepoproviders,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=products,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=environments,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;create;update;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups=hnc.x-k8s.io,resources=hierarchyconfigurations,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DeploymentRuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	dr := &v1alpha1.DeploymentRuntime{}
	err := r.Client.Get(ctx, req.NamespacedName, dr)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	task, err := r.Syncer.NewTasks(ctx, dr, dr.Status.DeployStatus)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("init runtime deployment task failed: %w", err)
	}

	if !dr.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(dr, runtimeFinalizerName) {
			return ctrl.Result{}, nil
		}

		cache, err := task.Delete(ctx)
		if err != nil {
			setDeployRuntimeStatus(dr, cache, nil, err)
			if err := r.Status().Update(ctx, dr); err != nil {
				logger.Error(err, errorMsgUpdateStatusFailed)
			}
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(dr, runtimeFinalizerName)
		if err := r.Update(ctx, dr); err != nil {
			return ctrl.Result{}, err
		}
		logger.V(1).Info("delete finish")
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(dr, runtimeFinalizerName) {
		controllerutil.AddFinalizer(dr, runtimeFinalizerName)
		if err := r.Update(ctx, dr); err != nil {
			return ctrl.Result{}, err
		}
	}

	illegalProjectRefs, err := dr.Validate(ctx, v1alpha1.NewValidateClientFromK8s(r.Client))
	if err != nil {
		setDeployRuntimeStatus(dr, nil, nil, err)
		if err := r.Status().Update(ctx, dr); err != nil {
			logger.Error(err, errorMsgUpdateStatusFailed)
		}
		return ctrl.Result{}, fmt.Errorf("validate runtime failed: %w", err)
	}

	if len(illegalProjectRefs) != 0 && len(illegalProjectRefs) == len(dr.Spec.ProjectsRef) {
		setDeployRuntimeStatus(dr, nil, illegalProjectRefs, fmt.Errorf("no valid project exists"))
		if err := r.Status().Update(ctx, dr); err != nil {
			logger.Error(err, errorMsgUpdateStatusFailed)
		}
		return ctrl.Result{RequeueAfter: reconcileFrequency}, nil
	}

	cache, err := task.Run(ctx)
	setDeployRuntimeStatus(dr, cache, nil, err)
	if err := r.Status().Update(ctx, dr); err != nil {
		return ctrl.Result{}, err
	}
	logger.V(1).Info("reconcile finish")

	return ctrl.Result{RequeueAfter: reconcileFrequency}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1alpha1.DeploymentRuntime{}, codeRepoMapField, func(rawObj client.Object) []string {
		deploymentRuntime := rawObj.(*v1alpha1.DeploymentRuntime)
		if deploymentRuntime.Spec.ManifestSource.CodeRepo == "" {
			return nil
		}
		return []string{deploymentRuntime.Spec.ManifestSource.CodeRepo}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DeploymentRuntime{}).
		Watches(
			&source.Kind{Type: &v1alpha1.CodeRepo{}},
			handler.EnqueueRequestsFromMapFunc(r.findDeploymentRuntimeForCoderepo),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *DeploymentRuntimeReconciler) findDeploymentRuntimeForCoderepo(coderepo client.Object) []reconcile.Request {
	attachedDeploymentRuntimes := &v1alpha1.DeploymentRuntimeList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(codeRepoMapField, coderepo.GetName()),
		Namespace:     coderepo.GetNamespace(),
	}
	err := r.List(context.TODO(), attachedDeploymentRuntimes, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedDeploymentRuntimes.Items))
	for i, item := range attachedDeploymentRuntimes.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func setDeployRuntimeStatus(runtime *v1alpha1.DeploymentRuntime, result *k8sRuntime.RawExtension, illegalProjectRefs []v1alpha1.IllegalProjectRef, err error) {
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
		runtime.Status.Conditions = v1alpha1.GetNewConditions(runtime.Status.Conditions, []metav1.Condition{condition}, map[string]bool{runtimeConditionType: true})
	} else {
		condition := metav1.Condition{
			Type:   runtimeConditionType,
			Status: "True",
			Reason: runtimeConditionReason,
		}
		runtime.Status.Conditions = v1alpha1.GetNewConditions(runtime.Status.Conditions, []metav1.Condition{condition}, map[string]bool{runtimeConditionType: true})
	}

	if len(illegalProjectRefs) != 0 {
		runtime.Status.IllegalProjectRefs = illegalProjectRefs
	} else {
		runtime.Status.IllegalProjectRefs = nil
	}
}
