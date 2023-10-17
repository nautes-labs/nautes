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
	"errors"
	"fmt"
	"time"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	warningFromBaseOperator                = "base-operator"
	clusterConditionTypeProductInfoUpdated = "ProductInfoUpdated"
	clusterConditionReason                 = "DataSourceChanged"
)

var (
	warningTypeProductNotFound warningType = "ProductNotFound"
	warningTypeProductNotMatch warningType = "ProductNotMatch"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cluster := &nautescrd.Cluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	if err != nil || !cluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cfg, err := nautescfg.NewNautesConfigFromFile()
	if err != nil {
		r.setCondition(cluster, err)
		if err := r.Status().Update(ctx, cluster); err != nil {
			logger.Error(err, "Update status failed.")
		}
		return ctrl.Result{}, err
	}
	updater := productUpdater{
		Client: r.Client,
		config: *cfg,
	}

	changed, err := updater.setProductsInfo(ctx, cluster)
	if changed {
		logger.V(1).Info("Product info has changed.")
	}
	r.setCondition(cluster, err)

	if err != nil || changed {
		if err := r.Status().Update(ctx, cluster); err != nil {
			logger.Error(err, "Update status failed.")
		}
	}

	logger.V(1).Info("Reconcile finished.")
	return ctrl.Result{RequeueAfter: time.Hour}, err
}

const (
	indexFieldProductsInCluster = "products"
	indexFieldProductName       = "productName"
)

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nautescrd.Cluster{}, indexFieldProductsInCluster, func(rawObj client.Object) []string {
		cluster := rawObj.(*nautescrd.Cluster)
		productMap := map[string]bool{}
		for _, namespace := range cluster.Spec.ReservedNamespacesAllowedProducts {
			for _, product := range namespace {
				productMap[product] = true
			}
		}

		products := []string{}
		for productName := range productMap {
			products = append(products, productName)
		}

		return products
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nautescrd.Product{}, indexFieldProductName, func(rawObj client.Object) []string {
		product := rawObj.(*nautescrd.Product)
		if product.Spec.Name == "" {
			return nil
		}
		return []string{product.Spec.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nautescrd.Cluster{}).
		Watches(
			&source.Kind{Type: &nautescrd.Product{}},
			handler.EnqueueRequestsFromMapFunc(r.findClusterForProduct),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *ClusterReconciler) findClusterForProduct(obj client.Object) []reconcile.Request {
	product := obj.(*nautescrd.Product)
	clusterList := &nautescrd.ClusterList{}
	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldProductsInCluster, product.Spec.Name),
	}
	if err := r.List(context.TODO(), clusterList, listOpts); err != nil {
		log.Log.Error(err, "List cluster by product failed.", "productName", product.Spec.Name)
		return nil
	}

	clusters := make([]reconcile.Request, len(clusterList.Items))
	for i := range clusterList.Items {
		clusters[i] = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&clusterList.Items[i])}
	}

	return clusters
}

func (r *ClusterReconciler) setCondition(cluster *nautescrd.Cluster, err error) {
	var condition metav1.Condition
	if err != nil {
		condition = metav1.Condition{
			Type:    clusterConditionTypeProductInfoUpdated,
			Status:  "False",
			Reason:  clusterConditionReason,
			Message: err.Error(),
		}
	} else {
		condition = metav1.Condition{
			Type:   clusterConditionTypeProductInfoUpdated,
			Status: "True",
			Reason: clusterConditionReason,
		}
	}
	cluster.Status.Conditions = nautescrd.GetNewConditions(cluster.Status.Conditions, []metav1.Condition{condition}, map[string]bool{clusterConditionTypeProductInfoUpdated: true})
}

type productUpdater struct {
	client.Client
	config nautescfg.Config
}

func (r *productUpdater) setProductsInfo(ctx context.Context, cluster *nautescrd.Cluster) (isChanged bool, err error) {
	logger := log.FromContext(ctx)
	isChanged = false

	oldWarningProducts := warningProducts{}
	otherWarnings := []nautescrd.Warning{}
	for _, warning := range cluster.Status.Warnings {
		if warning.From == warningFromBaseOperator {
			oldWarningProducts.add(warning.ID, warningType(warning.Type), nil)
		} else {
			otherWarnings = append(otherWarnings, warning)
		}
	}

	productIDMapisChanged, newWarningProducts, err := r.setProductIDMap(ctx, cluster)
	isChanged = isChanged || productIDMapisChanged
	if err != nil {
		return isChanged, err
	}

	if !newWarningProducts.IsSame(oldWarningProducts) {
		cluster.Status.Warnings = append(otherWarnings, newWarningProducts.GetWarnings()...)
		logger.V(1).Info("Warning message has changed.")
		isChanged = true
	}

	return isChanged, nil
}

func (r *productUpdater) setProductIDMap(ctx context.Context, cluster *nautescrd.Cluster) (isChanged bool, warnings warningProducts, err error) {
	logger := log.FromContext(ctx)
	warnings = warningProducts{}

	products := map[string]bool{}
	for _, namespace := range cluster.Spec.ReservedNamespacesAllowedProducts {
		for _, product := range namespace {
			products[product] = true
		}
	}

	for product := range cluster.Spec.ProductAllowedClusterResources {
		products[product] = true
	}

	for productName := range products {
		var warningError clusterUpdateError

		productID, ok := cluster.Status.ProductIDMap[productName]
		if !ok {
			product, err := r.getProductByName(ctx, productName)
			if err != nil {
				if errors.As(err, &warningError) {
					warnings.add(productName, warningError.GetWarningType(), warningError)
					logger.V(1).Info(err.Error())
					continue
				}
				return isChanged, warnings, err
			}

			if cluster.Status.ProductIDMap == nil {
				cluster.Status.ProductIDMap = map[string]string{}
			}
			cluster.Status.ProductIDMap[productName] = product.Name
			logger.V(1).Info("Product ID Map Updated.", "productName", productName)
			isChanged = true
			continue
		}

		if err := r.checkProductIDIsMatch(ctx, productName, productID); err != nil {
			if errors.As(err, &warningError) {
				warnings.add(productName, warningError.GetWarningType(), warningError)
				logger.V(1).Info(err.Error())
			} else {
				return isChanged, warnings, err
			}
		}
	}

	for productName := range cluster.Status.ProductIDMap {
		if !products[productName] {
			delete(cluster.Status.ProductIDMap, productName)
			logger.V(1).Info("Remove product from product ID map.", "productName", productName)
			isChanged = true
		}
	}

	return isChanged, warnings, nil
}

func (r *productUpdater) checkProductIDIsMatch(ctx context.Context, productName, productID string) error {
	product := &nautescrd.Product{}
	key := types.NamespacedName{
		Name:      productID,
		Namespace: r.config.Nautes.Namespace,
	}
	if err := r.Get(ctx, key, product); err != nil {
		if apierrors.IsNotFound(err) {
			return NewClusterUpdateErrorProductNotFound(productName)
		}
		return err
	}
	if productName != product.Spec.Name {
		return NewClusterUpdateErrorProductNotMatch(productName, productID)
	}
	return nil
}

func (r *productUpdater) getProductByName(ctx context.Context, productName string) (*nautescrd.Product, error) {
	productList := &nautescrd.ProductList{}
	listOpt := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexFieldProductName, productName),
	}
	if err := r.List(ctx, productList, listOpt); err != nil {
		return nil, err
	}

	products := []nautescrd.Product{}
	for _, product := range productList.Items {
		if !productList.Items[0].DeletionTimestamp.IsZero() {
			continue
		}
		products = append(products, product)
	}

	num := len(products)
	switch num {
	case 0:
		return nil, NewClusterUpdateErrorProductNotFound(productName)
	case 1:
		return &products[0], nil
	default:
		return nil, fmt.Errorf("obtain %d returns based on the project name %s", num, productName)
	}
}

type warningProducts map[string]warningProduct
type warningType string

type warningProduct struct {
	name        string
	warningType warningType
	error       error
}

func (p warningProducts) add(name string, warningType warningType, err error) {
	p[name] = warningProduct{
		name:        name,
		warningType: warningType,
		error:       err,
	}
}

func (p warningProducts) IsSame(products warningProducts) bool {
	if len(p) != len(products) {
		return false
	}

	for productName, productOne := range p {
		productTwo, ok := products[productName]
		if !ok ||
			productOne.name != productTwo.name ||
			productOne.warningType != productTwo.warningType {
			return false
		}
	}
	return true
}

func (p warningProducts) GetWarnings() []nautescrd.Warning {
	warnings := []nautescrd.Warning{}
	for _, product := range p {
		warnings = append(warnings, nautescrd.Warning{
			Type:    string(product.warningType),
			From:    warningFromBaseOperator,
			ID:      product.name,
			Message: product.error.Error(),
		})
	}
	return warnings
}

type clusterUpdateError struct {
	error
	warningType warningType
}

func NewClusterUpdateError(err error, warningType warningType) error {
	return clusterUpdateError{
		error:       err,
		warningType: warningType,
	}
}

func NewClusterUpdateErrorProductNotFound(productName string) error {
	return NewClusterUpdateError(
		fmt.Errorf("unable to find product %s in the system", productName),
		warningTypeProductNotFound)
}

func NewClusterUpdateErrorProductNotMatch(productName, productID string) error {
	return NewClusterUpdateError(
		fmt.Errorf("product ID %s does not match product name %s", productID, productName),
		warningTypeProductNotMatch,
	)
}

func (e clusterUpdateError) GetWarningType() warningType {
	return e.warningType
}
