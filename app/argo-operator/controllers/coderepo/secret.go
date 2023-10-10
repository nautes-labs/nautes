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
	"strconv"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	secret "github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/kops/pkg/kubeconfig"
)

const (
	SecretsEngine = "git"
	SecretsKey    = "deploykey"
)

type SecretContent struct {
	ID         string
	Kubeconfig *kubeconfig.KubectlConfig
	PrivateKey string
}

func (r *CodeRepoReconciler) getSecret(_ context.Context, codeRepo *resourcev1alpha1.CodeRepo, configs *nautesconfigs.Config) (*SecretContent, error) {
	secretsEngine := SecretsEngine
	secretsKey := SecretsKey
	secretPath := fmt.Sprintf("%s/%s/%s/%s", configs.Git.GitType, codeRepo.Name, "default", "readonly")
	secretOptions := secret.SecretOptions{
		SecretPath:   secretPath,
		SecretEngine: secretsEngine,
		SecretKey:    secretsKey,
	}

	secretConfig := &secret.SecretConfig{
		Namespace:  configs.Nautes.Namespace,
		SecretRepo: &configs.Secret,
	}

	if err := r.Secret.Init(secretConfig); err != nil {
		return nil, err
	}

	secretData, err := r.Secret.GetSecret(secretOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	return &SecretContent{
		ID:         strconv.Itoa(secretData.ID),
		PrivateKey: secretData.Data,
	}, nil
}

// isSecretChange Check if the code repository secret has changed
func isSecretChange(codeRepo *resourcev1alpha1.CodeRepo, id string) bool {
	if codeRepo.Status.Sync2ArgoStatus == nil {
		return false
	}

	if codeRepo.Status.Sync2ArgoStatus.SecretID == "" {
		return false
	}

	return id != codeRepo.Status.Sync2ArgoStatus.SecretID
}
