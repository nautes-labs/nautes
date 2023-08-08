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

package cluster

import (
	"context"
	"fmt"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	common "github.com/nautes-labs/nautes/app/argo-operator/controllers/common"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	utilString "github.com/nautes-labs/nautes/app/argo-operator/util/strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ResourceName  = "cluster name"
	FinalizerName = "argo.nautes.nautes.resource.nautes.io/finalizers"
)

var (
	namespacedName types.NamespacedName
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log                   logr.Logger
	Scheme                *runtime.Scheme
	Argocd                *argocd.ArgocdClient
	Secret                secret.SecretOperator
	GlobalConfigNamespace string
	GlobalConfigName      string
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters;deploymentruntimes;environments;projectpipelineruntimes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var requeueAfter = common.GetReconcileTime()

	namespacedName = req.NamespacedName
	cluster, err := r.getClusterResource(ctx, namespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Log.V(1).Info("Start Reconcile", "cluster Name", cluster.Name)

	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := cluster.ValidateCluster(ctx, cluster, r.Client, false); err != nil {
			r.Log.V(1).Error(err, "resource verification failed", ResourceName, cluster.Name)
			condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
			if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		if !utilString.ContainsString(cluster.ObjectMeta.Finalizers, FinalizerName) {
			if err := r.appendFinalizerAndUpdateStaus(ctx, FinalizerName, namespacedName); err != nil {
				r.Log.V(1).Error(err, "failed to add finalizer", ResourceName, cluster.Name, "finalizer", FinalizerName)
				condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
				if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}
		}
	} else {
		if cluster.Spec.ApiServer != "" {
			if err := r.deleteCluster(ctx, cluster.Spec.ApiServer); err != nil {
				errMsg := fmt.Errorf("failed to delete cluster when resource is deleted, err: %v", err)
				r.Log.V(1).Error(err, "failed to delete cluster when resource is deleted", ResourceName, cluster.Name)
				condition = metav1.Condition{Type: ClusterConditionType, Message: errMsg.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
				if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}
		}

		cluster.ObjectMeta.Finalizers = utilString.RemoveString(cluster.ObjectMeta.Finalizers, FinalizerName)
		if err := r.Update(context.Background(), cluster); err != nil {
			r.Log.V(1).Error(err, "failed to delete finalizer", ResourceName, cluster.Name)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	nautesConfigs, err := common.GetNautesConfigs(r.Client, r.GlobalConfigNamespace, r.GlobalConfigName)
	if err != nil {
		return ctrl.Result{}, err
	}

	secret, err := r.getSecret(ctx, req.Name, req.Namespace, nautesConfigs)
	if err != nil {
		r.Log.V(1).Error(err, "failed to get secret", ResourceName, cluster.Name)
		condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
		if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	tune, err := r.syncCluster2Argocd(ctx, cluster, secret.Kubeconfig, secret.ID)
	if tune && err != nil {
		errMsg := fmt.Errorf("failed to sync cluster to argocd, err: %v", err)
		r.Log.V(1).Error(err, "failed to sync cluster to argocd", ResourceName, cluster.Name)
		condition = metav1.Condition{Type: ClusterConditionType, Message: errMsg.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
		if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1alpha1.Cluster{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
