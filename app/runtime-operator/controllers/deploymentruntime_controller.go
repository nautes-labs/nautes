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

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
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

	secprovider "github.com/nautes-labs/nautes/app/runtime-operator/internal/secret/provider"

	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
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
	Scheme       *runtime.Scheme
	Syncer       interfaces.RuntimeSyncer
	NautesConfig nautescfg.NautesConfigs
}

var reconcileFrequency = time.Second * 60

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

	runtime := &nautescrd.DeploymentRuntime{}
	err := r.Client.Get(ctx, req.NamespacedName, runtime)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cfg, err := r.NautesConfig.GetConfigByClient(r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get nautes config failed: %w", err)
	}
	ctx = runtimecontext.NewNautesConfigContext(ctx, *cfg)

	secClient, err := secprovider.GetSecretClient(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get secret provider failed: %w", err)
	}
	ctx = runtimecontext.NewSecretClientContext(ctx, secClient)
	defer secClient.Logout()

	if !runtime.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(runtime, runtimeFinalizerName) {
			return ctrl.Result{}, nil
		}

		if runtime.Status.DeployHistory != nil {
			if err := r.Syncer.Delete(ctx, runtime); err != nil {
				setDeployRuntimeStatus(runtime, nil, nil, err)
				if err := r.Status().Update(ctx, runtime); err != nil {
					logger.Error(err, "update status failed")
				}
				return ctrl.Result{}, err
			}
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

	illegalProjectRefs, err := runtime.Validate(ctx, nautescrd.NewValidateClientFromK8s(r.Client))
	if err != nil {
		setDeployRuntimeStatus(runtime, nil, nil, err)
		if err := r.Status().Update(ctx, runtime); err != nil {
			logger.Error(err, "update status failed")
		}
		return ctrl.Result{}, fmt.Errorf("validate runtime failed: %w", err)
	}

	if len(illegalProjectRefs) != 0 && len(illegalProjectRefs) == len(runtime.Spec.ProjectsRef) {
		setDeployRuntimeStatus(runtime, nil, illegalProjectRefs, fmt.Errorf("No valid project exists"))
		if err := r.Status().Update(ctx, runtime); err != nil {
			logger.Error(err, "update status failed")
		}
		return ctrl.Result{RequeueAfter: reconcileFrequency}, nil
	}

	legalRuntime := NewDeploymentRuntimeWithOutIllegalProject(*runtime, illegalProjectRefs)
	deployInfo, err := r.Syncer.Sync(ctx, legalRuntime)

	setDeployRuntimeStatus(runtime, deployInfo, illegalProjectRefs, err)
	if err := r.Status().Update(ctx, runtime); err != nil {
		return ctrl.Result{}, err
	}
	logger.V(1).Info("reconcile finish")

	return ctrl.Result{RequeueAfter: reconcileFrequency}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentRuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nautescrd.DeploymentRuntime{}, codeRepoMapField, func(rawObj client.Object) []string {
		deploymentRuntime := rawObj.(*nautescrd.DeploymentRuntime)
		if deploymentRuntime.Spec.ManifestSource.CodeRepo == "" {
			return nil
		}
		return []string{deploymentRuntime.Spec.ManifestSource.CodeRepo}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nautescrd.DeploymentRuntime{}).
		Watches(
			&source.Kind{Type: &nautescrd.CodeRepo{}},
			handler.EnqueueRequestsFromMapFunc(r.findDeploymentRuntimeForCoderepo),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *DeploymentRuntimeReconciler) findDeploymentRuntimeForCoderepo(coderepo client.Object) []reconcile.Request {
	attachedDeploymentRuntimes := &nautescrd.DeploymentRuntimeList{}
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

func (r *DeploymentRuntimeReconciler) updateStatus() error {
	return nil
}

func NewDeploymentRuntimeWithOutIllegalProject(runtime nautescrd.DeploymentRuntime, illegalProjects []nautescrd.IllegalProjectRef) *nautescrd.DeploymentRuntime {
	newRuntime := runtime.DeepCopy().DeepCopyObject().(*nautescrd.DeploymentRuntime)
	// Set version to 0 to avoid syncer update resource by new runtime
	newRuntime.ResourceVersion = "0"

	if len(illegalProjects) == 0 {
		return newRuntime
	}

	indexIllegalProject := map[string]bool{}
	for _, project := range illegalProjects {
		indexIllegalProject[project.ProjectName] = true
	}

	newProjectRef := []string{}
	for _, project := range runtime.Spec.ProjectsRef {
		if !indexIllegalProject[project] {
			newProjectRef = append(newProjectRef, project)
		}
	}
	newRuntime.Spec.ProjectsRef = newProjectRef

	return newRuntime
}

func setDeployRuntimeStatus(runtime *nautescrd.DeploymentRuntime, result *interfaces.RuntimeDeploymentResult, illegalProjectRefs []nautescrd.IllegalProjectRef, err error) {
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

	if result != nil {
		runtime.Status.Cluster = result.Cluster
		if result.DeploymentDeploymentResult != nil {
			runtime.Status.DeployHistory = &nautescrd.DeployHistory{
				ManifestSource: runtime.Spec.ManifestSource,
				Destination:    runtime.Spec.Destination,
				Source:         result.DeploymentDeploymentResult.Source,
			}
		}
	}

	if len(illegalProjectRefs) != 0 {
		runtime.Status.IllegalProjectRefs = illegalProjectRefs
	} else {
		runtime.Status.IllegalProjectRefs = nil
	}
}
