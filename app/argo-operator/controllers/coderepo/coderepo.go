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

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
)

type CodeRepoParams struct {
	url                      string
	repoName                 string
	secret                   *SecretContent
	skipRepositoryValidCheck bool
}

func NewCodeRepoParams(url, repoName string, secret *SecretContent) *CodeRepoParams {
	return &CodeRepoParams{
		url:      url,
		repoName: repoName,
		secret:   secret,
	}
}

func (r *CodeRepoReconciler) saveCodeRepo(params *CodeRepoParams) (bool, error) {
	codeRepoInfo, err := r.getRepository(params.url)
	if err != nil {
		_, ok := err.(*argocd.ErrorNotFound)
		if !ok {
			return false, err
		}
		if err = r.createRepository(context.Background(), params.repoName, params.secret.PrivateKey, params.url); err != nil {
			return false, err
		}
		return true, nil
	}

	if params.skipRepositoryValidCheck || !codeRepoInfo.valid {
		if err = r.updateRepository(context.Background(), params.repoName, params.secret.PrivateKey, params.url); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *CodeRepoReconciler) deleteCodeRepo(url string) error {
	_, err := r.getRepository(url)
	if err != nil {
		_, ok := err.(*argocd.ErrorNotFound)
		if !ok {
			return err
		}
	} else {
		if err := r.deleteRepository(url); err != nil {
			return err
		}
	}
	return nil
}

// isCodeRepoChange Check if the code repository url has changed
func isCodeRepoChange(codeRepo *resourcev1alpha1.CodeRepo, url string) (string, bool) {
	if codeRepo.Status.Sync2ArgoStatus != nil && codeRepo.Status.Sync2ArgoStatus.Url != "" && codeRepo.Status.Sync2ArgoStatus.Url != url {
		return codeRepo.Status.Sync2ArgoStatus.Url, true
	}

	return "", false
}
