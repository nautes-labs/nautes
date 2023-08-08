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
	"net/url"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetCodeRepoURL(ctx context.Context, k8sClient client.Client, codeRepo nautescrd.CodeRepo, nautesNamespace string) (string, error) {
	if codeRepo.Spec.URL != "" {
		return codeRepo.Spec.URL, nil
	}

	provider := &nautescrd.CodeRepoProvider{}
	key := types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      codeRepo.Spec.CodeRepoProvider,
	}
	if err := k8sClient.Get(ctx, key, provider); err != nil {
		return "", err
	}

	codeRepoBaseURL := provider.Spec.SSHAddress

	product := &nautescrd.Product{}
	key = types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      codeRepo.Spec.Product,
	}
	if err := k8sClient.Get(ctx, key, product); err != nil {
		return "", err
	}
	codeRepoGroupPath := product.Spec.Name

	return url.JoinPath(codeRepoBaseURL, codeRepoGroupPath, fmt.Sprintf("%s.git", codeRepo.Spec.RepoName))
}
