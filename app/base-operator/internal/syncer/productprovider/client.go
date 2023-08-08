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

package productprovider

import (
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	coderepoprovider "github.com/nautes-labs/nautes/app/base-operator/internal/coderepo/provider"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautesctx "github.com/nautes-labs/nautes/pkg/context"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	ProductProviders["coderepo"] = coderepoprovider.NewProductProviderCodeRepo()
}

const (
	CONTEXT_KEY_LABEL        nautesctx.ContextKey = "productprovider.syncer.label"
	CONTEXT_KEY_CFG          nautesctx.ContextKey = "productprovider.syncer.config"
	CONTEXT_KEY_LIST_OPTIONS nautesctx.ContextKey = "productprovider.syncer.listopts"
)

var ProductProviders = map[string]ProviderFacotry{}

type ProductProviderSyncer struct {
	client       client.Client
	NautesConfig nautescfg.NautesConfigs
	Rest         *rest.Config
}

func (s *ProductProviderSyncer) Setup() error {
	err := nautescrd.AddToScheme(scheme.Scheme)
	if err != nil {
		return err
	}

	k8sClient, err := client.New(s.Rest, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}
	s.client = k8sClient

	return nil
}

func (s *ProductProviderSyncer) Sync(ctx context.Context, productProvider nautescrd.ProductProvider) error {
	cfg, err := s.NautesConfig.GetConfigByRest(s.Rest)
	if err != nil {
		return fmt.Errorf("get nautes configs failed: %w", err)
	}
	label := map[string]string{nautescrd.LABEL_FROM_PRODUCT_PROVIDER: productProvider.Name}
	listOpts := []client.ListOption{
		client.MatchingLabels(label),
		client.InNamespace(productProvider.Namespace),
	}

	ctx = NewSyncContext(ctx, *cfg, label, listOpts)

	source, err := GetProvider(ctx, CONTEXT_KEY_CFG, s.client)
	if err != nil {
		return fmt.Errorf("get product provider failed: %w", err)
	}
	sourceProducts, err := source.GetProducts()
	if err != nil {
		return fmt.Errorf("get source product list failed: %w", err)
	}

	k8sProducts := &nautescrd.ProductList{}
	err = s.client.List(ctx, k8sProducts, listOpts...)
	if err != nil {
		return fmt.Errorf("get k8s product list failed: %w", err)
	}

	newList, updateList, deleteList, err := compareProduct(sourceProducts, k8sProducts.Items)
	if err != nil {
		return fmt.Errorf("get error in compare product: %w", err)
	}
	errs := []error{}
	errs = append(errs, s.createProduct(ctx, newList, &productProvider)...)
	errs = append(errs, s.updateProduct(ctx, updateList, &productProvider)...)
	errs = append(errs, s.deleteProduct(ctx, deleteList)...)
	if len(errs) != 0 {
		return fmt.Errorf("get error in sync product: %v", errs)
	}

	return nil
}

// compareProduct check product in provider list is also in k8s list, if yes , check it is need to update.
// then check product in k8s list is not in provider list, finaly get product need to create update delete
func compareProduct(srcProducts, k8sProducts []nautescrd.Product) ([]nautescrd.Product, []nautescrd.Product, []nautescrd.Product, error) {
	newList := []nautescrd.Product{}
	updateList := []nautescrd.Product{}
	deleteList := []nautescrd.Product{}

	for _, srcProduct := range srcProducts {
		isNew := true
		for i, k8sProduct := range k8sProducts {
			if srcProduct.Name == k8sProduct.Name {
				isNew = false
				if !isSameProduct(srcProduct.Spec, k8sProduct.Spec) {
					updateList = append(updateList, srcProduct)
				}
				k8sProducts = append(k8sProducts[:i], k8sProducts[i+1:]...)
				break
			}
		}
		if isNew {
			newList = append(newList, srcProduct)
		}
	}

	deleteList = k8sProducts
	return newList, updateList, deleteList, nil
}

func isSameProduct(new, old nautescrd.ProductSpec) bool {
	if new.Name == old.Name &&
		new.MetaDataPath == old.MetaDataPath {
		return true
	}
	return false
}

func (s *ProductProviderSyncer) createProduct(ctx context.Context, products []nautescrd.Product, provider *nautescrd.ProductProvider) []error {
	errs := []error{}

	_, label, _, err := FromSyncContext(ctx)
	if err != nil {
		return append(errs, err)
	}

	for _, product := range products {
		product.Namespace = provider.Namespace
		product.Labels = label
		err := s.client.Create(ctx, &product)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (s *ProductProviderSyncer) updateProduct(ctx context.Context, products []nautescrd.Product, productProvier *nautescrd.ProductProvider) []error {
	errs := []error{}
	for _, product := range products {
		tmp := &nautescrd.Product{}
		key := types.NamespacedName{
			Namespace: productProvier.Namespace,
			Name:      product.Name,
		}

		err := s.client.Get(ctx, key, tmp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		tmp.Spec = *product.Spec.DeepCopy()
		err = s.client.Update(ctx, tmp)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (s *ProductProviderSyncer) deleteProduct(ctx context.Context, products []nautescrd.Product) []error {
	errs := []error{}
	for _, product := range products {
		err := s.client.Delete(ctx, &product)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func GetProvider(ctx context.Context, key nautesctx.ContextKey, client client.Client) (baseinterface.ProductProvider, error) {
	cfg, err := nautesctx.FromConfigContext(ctx, key)
	if err != nil {
		return nil, err
	}

	providers := &nautescrd.ProductProviderList{}
	if err = client.List(ctx, providers); err != nil {
		return nil, fmt.Errorf("get resource product provider failed: %w", err)
	}

	if len(providers.Items) != 1 {
		return nil, fmt.Errorf("product provider is not only")
	}

	provider := providers.Items[0]
	productProviderFactory, ok := ProductProviders[provider.Spec.Type]
	if !ok {
		return nil, fmt.Errorf("unknow provider type")
	}
	prodcutProvider, err := productProviderFactory.GetProvider(ctx, provider.Spec.Name, client, *cfg)
	if err != nil {
		return nil, fmt.Errorf("get product provider failed: %w", err)
	}

	return prodcutProvider, nil
}

func NewSyncContext(ctx context.Context, cfg nautescfg.Config, label map[string]string, listOpts []client.ListOption) context.Context {
	ctx = context.WithValue(ctx, CONTEXT_KEY_CFG, cfg)
	ctx = context.WithValue(ctx, CONTEXT_KEY_LABEL, label)
	return context.WithValue(ctx, CONTEXT_KEY_LIST_OPTIONS, listOpts)
}

func FromSyncContext(ctx context.Context) (*nautescfg.Config, map[string]string, []client.ListOption, error) {
	inter := ctx.Value(CONTEXT_KEY_LIST_OPTIONS)
	listOpts, ok := inter.([]client.ListOption)
	if !ok {
		return nil, nil, nil, fmt.Errorf("get list options failed")
	}
	inter = ctx.Value(CONTEXT_KEY_LABEL)
	label, ok := inter.(map[string]string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("get label failed")
	}
	inter = ctx.Value(CONTEXT_KEY_CFG)
	cfg, ok := inter.(nautescfg.Config)
	if !ok {
		return nil, nil, nil, fmt.Errorf("get cfg failed")
	}
	return &cfg, label, listOpts, nil
}
