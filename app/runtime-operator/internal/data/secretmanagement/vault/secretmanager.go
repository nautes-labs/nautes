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

package vault

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	vaultproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
)

type codeRepoManager struct {
	vaultproxy.AuthGrantHTTPClient
	clusterName string
}

// GrantPermission grants the machine account access to the secret of code repo type.
func (c *codeRepoManager) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	if repo.CodeRepo == nil {
		return fmt.Errorf("coderepo info is missing")
	}

	req := c.getCodeRepoRequest(account, repo)
	_, err := c.GrantAuthroleGitPolicy(ctx, req)
	return err
}

// getCodeRepoRequest builds a request body of the secret of git repo type for the Vault proxy.
func (c *codeRepoManager) getCodeRepoRequest(account component.MachineAccount, repo component.SecretInfo) *vaultproxy.AuthroleGitPolicyRequest {
	req := &vaultproxy.AuthroleGitPolicyRequest{
		ClusterName: c.clusterName,
		DestUser:    account.Name,
		Secret: &vaultproxy.GitMeta{
			ProviderType: repo.CodeRepo.ProviderType,
			Id:           repo.CodeRepo.ID,
			Username:     repo.CodeRepo.User,
			Permission:   string(repo.CodeRepo.Permission),
		},
	}
	return req
}

// RevokePermission revokes the machine account access to the secret of code repo type.
func (c *codeRepoManager) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	if repo.CodeRepo == nil {
		return fmt.Errorf("coderepo info is missing")
	}

	req := c.getCodeRepoRequest(account, repo)
	_, err := c.RevokeAuthroleGitPolicy(ctx, req)
	return err
}

type artifactRepoManager struct {
	vaultproxy.AuthGrantHTTPClient
	clusterName string
}

// GrantPermission grants the machine account access to the secret of artifact repo type.
func (a *artifactRepoManager) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	if repo.ArtifactAccount == nil {
		return fmt.Errorf("artifact account info is missing")
	}
	req := a.getRepoAccountRequest(account, repo)
	_, err := a.GrantAuthroleRepoPolicy(ctx, req)
	return err
}

// RevokePermission revokes the machine account access to the secret of artifact repo type.
func (a *artifactRepoManager) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	if repo.ArtifactAccount == nil {
		return fmt.Errorf("artifact account info is missing")
	}
	req := a.getRepoAccountRequest(account, repo)
	_, err := a.RevokeAuthroleRepoPolicy(ctx, req)
	return err
}

// getRepoAccountRequest builds a request body of the secret of artifact repo type for the Vault proxy.
func (a *artifactRepoManager) getRepoAccountRequest(account component.MachineAccount, repo component.SecretInfo) *vaultproxy.AuthroleRepoPolicyRequest {
	req := &vaultproxy.AuthroleRepoPolicyRequest{
		ClusterName: a.clusterName,
		DestUser:    account.Name,
		Secret: &vaultproxy.RepoMeta{
			ProviderId: repo.ArtifactAccount.ProviderName,
			Product:    repo.ArtifactAccount.Product,
			Project:    repo.ArtifactAccount.Project,
		},
	}
	return req
}
