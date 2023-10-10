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

package coderepo

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	common "github.com/nautes-labs/nautes/app/argo-operator/controllers/common"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ResourceName  = "codeRepo name"
	FinalizerName = "argo.nautes.resource.nautes.io/finalizers"
)

// CodeRepoReconciler reconciles a CodeRepo object
type CodeRepoReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	Argocd                *argocd.ArgocdClient
	Secret                secret.SecretOperator
	Log                   logr.Logger
	URL                   string
	GlobalConfigNamespace string
	GlobalConfigName      string
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes;products;coderepoproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

func (r *CodeRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var requeueAfter = common.GetReconcileTime()
	var codeRepo = &resourcev1alpha1.CodeRepo{}

	err := r.Get(ctx, req.NamespacedName, codeRepo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.V(1).Info("Start Reconcile", ResourceName, codeRepo.Name)
	if codeRepo.ObjectMeta.DeletionTimestamp.IsZero() {
		if err = codeRepo.Validate(); err != nil {
			r.Log.V(1).Error(err, "resource verification failed", ResourceName, codeRepo.Name)
			if err := r.setConditionAndUpdateStatus(ctx, codeRepo, err.Error(), metav1.ConditionFalse); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfter}, nil
			}
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		if err := r.AddFinalizerAndUpdateStatus(ctx, codeRepo, FinalizerName); err != nil {
			r.Log.V(1).Error(err, "failed to add finalizer", ResourceName, codeRepo.Name)
			return ctrl.Result{}, err
		}
	} else {
		if codeRepo.Status.Sync2ArgoStatus != nil && codeRepo.Status.Sync2ArgoStatus.Url != "" {
			if err := r.deleteCodeRepo(codeRepo.Status.Sync2ArgoStatus.Url); err != nil {
				errMsg := fmt.Errorf("failed to delete repository when resource is deleted, err: %v", err)
				r.Log.V(1).Error(err, "failed to delete repository when resource is deleted", ResourceName, codeRepo.Name)
				if err := r.setConditionAndUpdateStatus(ctx, codeRepo, errMsg.Error(), metav1.ConditionFalse); err != nil {
					return ctrl.Result{}, err
				}

				return ctrl.Result{}, err
			}
		}

		r.Log.V(1).Info("Successfully delete repository to argocd", ResourceName, codeRepo.Name)

		if err := r.DeleteFinalizerAndUpdateStatus(ctx, codeRepo, FinalizerName); err != nil {
			r.Log.V(1).Error(err, "failed to delete finalizer", ResourceName, codeRepo.Name)
			return ctrl.Result{}, err
		}

		r.Log.V(1).Info("successfully delete codeRepo", ResourceName, codeRepo.Name)
		return ctrl.Result{}, nil
	}

	url, err := codeRepo.GetURL(codeRepo.Spec)
	if err != nil {
		errMsg := fmt.Errorf("failed to get codeRepo url, err: %v", err)
		r.Log.V(1).Error(err, "failed to get codeRepo url", ResourceName, codeRepo.Name)
		if err := r.setConditionAndUpdateStatus(ctx, codeRepo, errMsg.Error(), metav1.ConditionFalse); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	r.Log.V(1).Info("Successfully get codeRepo url", ResourceName, codeRepo.Name, "url", url)

	nautesConfigs, err := common.GetNautesConfigs(r.Client, r.GlobalConfigNamespace, r.GlobalConfigName)
	if err != nil {
		return ctrl.Result{}, err
	}

	secretData, err := r.getSecret(ctx, codeRepo, nautesConfigs)
	if err != nil {
		errMsg := fmt.Errorf("failed to get secret, err: %v", err)
		r.Log.V(1).Error(err, "failed to get secret", ResourceName, codeRepo.Name)
		if err := r.setConditionAndUpdateStatus(ctx, codeRepo, errMsg.Error(), metav1.ConditionFalse); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	r.Log.V(1).Info("Successfully get secret", ResourceName, codeRepo.Name)

	sync, err := r.syncCodeRepo2Argocd(codeRepo, url, secretData)
	if err != nil {
		errMsg := fmt.Errorf("failed to sync codeRepo to argocd, err: %v", err)
		r.Log.V(1).Error(err, "failed to sync codeRepo to argocd", ResourceName, codeRepo.Name)
		if err := r.setConditionAndUpdateStatus(ctx, codeRepo, errMsg.Error(), metav1.ConditionFalse); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if sync {
		if err = r.setSync2ArgoStatus(ctx, codeRepo, url, secretData.ID); err != nil {
			r.Log.V(1).Error(err, "failed to set Sync2ArgoStatus status", ResourceName, codeRepo.Name)
			if err := r.setConditionAndUpdateStatus(ctx, codeRepo, err.Error(), metav1.ConditionFalse); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}

		message := fmt.Sprintf("successfully save %s to argocd", codeRepo.Name)
		if err := r.setConditionAndUpdateStatus(ctx, codeRepo, message, metav1.ConditionTrue); err != nil {
			return ctrl.Result{}, err
		}

		r.Log.V(1).Info(message, ResourceName, codeRepo.Name)
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CodeRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1alpha1.CodeRepo{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
