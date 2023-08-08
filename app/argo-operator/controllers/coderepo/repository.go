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

import "context"

type codeRepoInfo struct {
	name  string
	repo  string
	valid bool
}

func (r *CodeRepoReconciler) getRepository(url string) (*codeRepoInfo, error) {
	err := r.Argocd.Auth().Login()
	if err != nil {
		return nil, err
	}

	response, err := r.Argocd.CodeRepo().GetRepositoryInfo(url)
	if err != nil {
		return nil, err
	}

	valid := response.ConnectionState.Status == "Successful" || response.ConnectionState.Status == "Unknown"

	return &codeRepoInfo{
		name:  response.Name,
		repo:  response.Repo,
		valid: valid,
	}, nil
}

func (r *CodeRepoReconciler) updateRepository(ctx context.Context, repoName, secret, url string) error {
	err := r.Argocd.Auth().Login()
	if err != nil {
		return err
	}

	err = r.Argocd.CodeRepo().UpdateRepository(repoName, secret, url)
	if err != nil {
		return err
	}

	return nil
}

func (r *CodeRepoReconciler) createRepository(ctx context.Context, repoName, secret, url string) error {
	if err := r.Argocd.Auth().Login(); err != nil {
		return err
	}

	if err := r.Argocd.CodeRepo().CreateRepository(repoName, secret, url); err != nil {
		return err
	}

	return nil
}

func (r *CodeRepoReconciler) deleteRepository(url string) error {
	if err := r.Argocd.Auth().Login(); err != nil {
		return err
	}

	if err := r.Argocd.CodeRepo().DeleteRepository(url); err != nil {
		return err
	}

	return nil
}
