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

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	productFinalizerName = "product.base-operator.nautes.resource.nautes.io/finalizers"
)

var (
	productConditionType   = "ProductReady"
	productConditionReason = "RegularUpdate"
)

// ProductReconciler reconciles a Product object
type ProductReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Syncer syncer.ProductSyncer
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=products,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=products/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=products/finalizers,verbs=update
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list;create;update
//+kubebuilder:rbac:groups=argoproj.io,resources=appprojects,verbs=get;create;update;patch
//+kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ProductReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	product := &nautescrd.Product{}
	if err := r.Get(ctx, req.NamespacedName, product); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !product.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(product, productFinalizerName) {
			return ctrl.Result{}, nil
		}

		if err := r.Syncer.Delete(ctx, *product); err != nil {
			r.setStatus(product, err)
			if err := r.Status().Update(ctx, product); err != nil {
				logger.Error(err, "update status failed")
			}
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(product, productFinalizerName)
		if err := r.Update(ctx, product); err != nil {
			return ctrl.Result{}, err
		}
		logger.V(1).Info("delete finish")
		return ctrl.Result{}, nil
	}

	controllerutil.AddFinalizer(product, productFinalizerName)
	if err := r.Update(ctx, product); err != nil {
		return ctrl.Result{}, err
	}

	err := r.Syncer.Sync(ctx, *product)
	r.setStatus(product, err)
	if err := r.Status().Update(ctx, product); err != nil {
		logger.Error(err, "update status failed")
	}

	logger.V(1).Info("sync finish")
	return ctrl.Result{}, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *ProductReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nautescrd.Product{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

func (r *ProductReconciler) setStatus(product *nautescrd.Product, err error) {
	if err != nil {
		condition := metav1.Condition{
			Type:    productConditionType,
			Status:  "False",
			Reason:  productConditionReason,
			Message: err.Error(),
		}
		product.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{productConditionType: true})
	} else {
		condition := metav1.Condition{
			Type:   productConditionType,
			Status: "True",
			Reason: productConditionReason,
		}
		product.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{productConditionType: true})
	}
}
