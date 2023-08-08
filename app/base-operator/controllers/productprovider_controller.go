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
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	productProviderFinalizerName = "productprovider.base-operator.nautes.resource.nautes.io/finalizers"
)

var (
	productProviderConditionType   = "ProductListSynced"
	productProviderConditionReason = "RegularUpdate"
)

// ProductProviderReconciler reconciles a ProductProvider object
type ProductProviderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Syncer syncer.ProductProviderSyncer
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=productproviders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=productproviders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=productproviders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ProductProviderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	productProvider := &nautescrd.ProductProvider{}
	if err := r.Get(ctx, req.NamespacedName, productProvider); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !productProvider.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(productProvider, productProviderFinalizerName) {
			return ctrl.Result{}, nil
		}

		opts := []client.ListOption{
			client.MatchingLabels(map[string]string{nautescrd.LABEL_FROM_PRODUCT_PROVIDER: productProvider.Name}),
			client.InNamespace(productProvider.Namespace),
		}

		if err := nautescrd.ProductProviderIsDeletable(ctx, r.Client, opts...); err != nil {
			r.setStatus(productProvider, err)
			if err := r.Status().Update(ctx, productProvider); err != nil {
				logger.Error(err, "update status failed")
			}
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(productProvider, productProviderFinalizerName)
		if err := r.Update(ctx, productProvider); err != nil {
			return ctrl.Result{}, err
		}
		logger.V(1).Info("delete finish")
		return ctrl.Result{}, nil
	}

	controllerutil.AddFinalizer(productProvider, productProviderFinalizerName)
	if err := r.Update(ctx, productProvider); err != nil {
		return ctrl.Result{}, err
	}

	err := r.Syncer.Sync(ctx, *productProvider)
	r.setStatus(productProvider, err)
	if err := r.Status().Update(ctx, productProvider); err != nil {
		logger.Error(err, "update status failed")
	}

	logger.V(1).Info("sync finish")
	return ctrl.Result{RequeueAfter: time.Second * 60}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProductProviderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nautescrd.ProductProvider{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *ProductProviderReconciler) setStatus(provider *nautescrd.ProductProvider, err error) {
	if err != nil {
		condition := metav1.Condition{
			Type:    productProviderConditionType,
			Status:  "False",
			Reason:  productProviderConditionReason,
			Message: err.Error(),
		}
		provider.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{productProviderConditionType: true})
	} else {
		condition := metav1.Condition{
			Type:   productProviderConditionType,
			Status: "True",
			Reason: productProviderConditionReason,
		}
		provider.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{productProviderConditionType: true})
	}
}
