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

package utils

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCodeRepoProviderAndCodeRepoWithURL(ctx context.Context, k8sClient client.Client, repoKey types.NamespacedName, nautesNamespace string) (*nautescrd.CodeRepoProvider, *nautescrd.CodeRepo, error) {
	codeRepo := &nautescrd.CodeRepo{}
	if err := k8sClient.Get(ctx, repoKey, codeRepo); err != nil {
		return nil, nil, fmt.Errorf("get code repo faile: %w", err)
	}

	provider := &nautescrd.CodeRepoProvider{}
	providerKey := types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      codeRepo.Spec.CodeRepoProvider,
	}
	if err := k8sClient.Get(ctx, providerKey, provider); err != nil {
		return nil, nil, fmt.Errorf("get provider failed: %w", err)
	}

	if codeRepo.Spec.URL != "" {
		return provider, codeRepo, nil
	}

	if codeRepo.Spec.RepoName == "" {
		return nil, nil, fmt.Errorf("repo name is empty")
	}

	product := &nautescrd.Product{}
	productKey := types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      codeRepo.Spec.Product,
	}
	if err := k8sClient.Get(ctx, productKey, product); err != nil {
		return nil, nil, fmt.Errorf("get product failed: %w", err)
	}

	codeRepo.Spec.URL = fmt.Sprintf("%s/%s/%s", provider.Spec.SSHAddress, product.Spec.Name, codeRepo.Spec.RepoName)

	return provider, codeRepo, nil
}
