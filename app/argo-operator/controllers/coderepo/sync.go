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
	"fmt"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

// syncCodeRepo2Argocd returns boolean value of true expected update status
func (r *CodeRepoReconciler) syncCodeRepo2Argocd(codeRepo *resourcev1alpha1.CodeRepo, url string, secret *SecretContent) (bool, error) {
	params := NewCodeRepoParams(url, codeRepo.Name, secret)

	if preCodeRepoURL, ok := isCodeRepoChange(codeRepo, url); ok {
		r.Log.V(1).Info("the codeRepo %s url is change", codeRepo.Name)

		err := r.deleteCodeRepo(preCodeRepoURL)
		if err != nil {
			return false, err
		}

		sync, err := r.saveCodeRepo(params)
		if err != nil {
			return false, fmt.Errorf("failed to save repository when url change, err: %w", err)
		}

		return sync, nil
	}

	if ok := isSecretChange(codeRepo, secret.ID); ok {
		r.Log.V(1).Info("the codeRepo %s secret version is change", codeRepo.Name)

		params.skipRepositoryValidCheck = true
		sync, err := r.saveCodeRepo(params)
		if err != nil {
			return false, fmt.Errorf("failed to save repository when secret change, err: %w", err)
		}

		return sync, nil
	}

	sync, err := r.saveCodeRepo(params)
	if err != nil {
		return false, fmt.Errorf("failed to save repository, err: %w", err)
	}

	return sync, nil
}
