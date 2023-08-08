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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var productproviderlog = logf.Log.WithName("productprovider-resource")

func (r *ProductProvider) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nautes-resource-nautes-io-v1alpha1-productprovider,mutating=false,failurePolicy=fail,sideEffects=None,groups=nautes.resource.nautes.io,resources=productproviders,verbs=delete,versions=v1alpha1,name=vproductprovider.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ProductProvider{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProductProvider) ValidateCreate() error {
	productproviderlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProductProvider) ValidateUpdate(old runtime.Object) error {
	productproviderlog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProductProvider) ValidateDelete() error {
	productproviderlog.Info("validate delete", "name", r.Name)

	k8sClient, err := getClient()
	if err != nil {
		return err
	}

	return r.IsDeletable(context.TODO(), k8sClient)
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=products,verbs=get;list

func (r *ProductProvider) IsDeletable(ctx context.Context, k8sClient client.Client) error {
	products := &ProductList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{LABEL_FROM_PRODUCT_PROVIDER: r.Name}),
		client.InNamespace(r.Namespace),
	}
	err := k8sClient.List(ctx, products, listOpts...)
	if err != nil {
		return fmt.Errorf("list product failed: %w", err)
	}
	if len(products.Items) != 0 {
		return fmt.Errorf("products is not none")
	}
	return nil
}

func ProductProviderIsDeletable(ctx context.Context, k8sClient client.Client, opts ...client.ListOption) error {
	products := &ProductList{}
	err := k8sClient.List(ctx, products, opts...)
	if err != nil {
		return fmt.Errorf("list product failed: %w", err)
	}
	if len(products.Items) != 0 {
		return fmt.Errorf("products is not none")
	}
	return nil
}
