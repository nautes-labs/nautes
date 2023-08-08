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

package coderepoprovider

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	gitlab "github.com/nautes-labs/nautes/app/base-operator/internal/coderepo/gitlab"
	secretprovider "github.com/nautes-labs/nautes/app/base-operator/internal/secret/provider"
	baseinterface "github.com/nautes-labs/nautes/app/base-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	ProductProviderCodeRepoFactory["gitlab"] = gitlab.NewProvider
}

type NewProductProviderCoderRepo func(token string, codeRepoProvider nautescrd.CodeRepoProvider, cfg nautescfg.Config) (baseinterface.ProductProvider, error)

var ProductProviderCodeRepoFactory = map[string]NewProductProviderCoderRepo{}

// ProductProviderCodeRepo is used to generate the coderepo type of product provider.
type ProductProviderCodeRepo struct {
	ProviderFactory map[string]NewProductProviderCoderRepo
}

func NewProductProviderCodeRepo() *ProductProviderCodeRepo {
	return &ProductProviderCodeRepo{
		ProviderFactory: ProductProviderCodeRepoFactory,
	}
}

func (p *ProductProviderCodeRepo) GetProvider(ctx context.Context, codeRepoProviderName string, k8sClient client.Client, cfg nautescfg.Config) (baseinterface.ProductProvider, error) {
	provider := &nautescrd.CodeRepoProvider{}
	key := types.NamespacedName{
		Namespace: cfg.Nautes.Namespace,
		Name:      codeRepoProviderName,
	}
	err := k8sClient.Get(ctx, key, provider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)

	}
	NewProvider, ok := p.ProviderFactory[provider.Spec.ProviderType]
	if !ok {
		return nil, fmt.Errorf("unknow code repo provider type")
	}

	secClient, err := secretprovider.GetSecretStore(ctx, cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("get secret provider failed: %w", err)
	}
	defer secClient.Logout()

	token, err := secClient.GetGitRepoRootToken(ctx, provider.Name)
	if err != nil {
		return nil, fmt.Errorf("get root token failed: %w", err)
	}

	productProvider, err := NewProvider(token, *provider, cfg)
	if err != nil {
		return nil, fmt.Errorf("get %s failed: %w", provider.Spec.ProviderType, err)
	}
	return productProvider, nil
}
