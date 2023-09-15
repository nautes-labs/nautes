package vault

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	vaultproxy "github.com/nautes-labs/nautes/pkg/client/vaultproxy"
)

type codeRepoManager struct {
	vaultproxy.AuthGrantHTTPClient
	clusterName string
}

func (c *codeRepoManager) GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	if repo.CodeRepo == nil {
		return fmt.Errorf("coderepo info is missing")
	}

	req := &vaultproxy.AuthroleGitPolicyRequest{
		ClusterName: c.clusterName,
		DestUser:    user.Name,
		Secret: &vaultproxy.GitMeta{
			ProviderType: repo.CodeRepo.ProviderType,
			Id:           repo.CodeRepo.ID,
			Username:     repo.CodeRepo.User,
			Permission:   string(repo.CodeRepo.Permission),
		},
	}
	_, err := c.GrantAuthroleGitPolicy(ctx, req)
	return err
}

func (c *codeRepoManager) RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	if repo.CodeRepo == nil {
		return fmt.Errorf("coderepo info is missing")
	}

	req := &vaultproxy.AuthroleGitPolicyRequest{
		ClusterName: c.clusterName,
		DestUser:    user.Name,
		Secret: &vaultproxy.GitMeta{
			ProviderType: repo.CodeRepo.ProviderType,
			Id:           repo.CodeRepo.ID,
			Username:     repo.CodeRepo.User,
			Permission:   string(repo.CodeRepo.Permission),
		},
	}
	_, err := c.RevokeAuthroleGitPolicy(ctx, req)
	return err
}

type artifactRepoManager struct {
	vaultproxy.AuthGrantHTTPClient
	clusterName string
}

func (a *artifactRepoManager) GrantPermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	if repo.ArtifactAccount == nil {
		return fmt.Errorf("artifact account info is missing")
	}
	req := &vaultproxy.AuthroleRepoPolicyRequest{
		ClusterName: a.clusterName,
		DestUser:    user.Name,
		Secret: &vaultproxy.RepoMeta{
			ProviderId: repo.ArtifactAccount.ProviderName,
			Product:    repo.ArtifactAccount.Product,
			Project:    repo.ArtifactAccount.Project,
		},
	}
	_, err := a.GrantAuthroleRepoPolicy(ctx, req)
	return err
}

func (a *artifactRepoManager) RevokePermission(ctx context.Context, repo syncer.SecretInfo, user syncer.User) error {
	if repo.ArtifactAccount == nil {
		return fmt.Errorf("artifact account info is missing")
	}
	req := &vaultproxy.AuthroleRepoPolicyRequest{
		ClusterName: a.clusterName,
		DestUser:    user.Name,
		Secret: &vaultproxy.RepoMeta{
			ProviderId: repo.ArtifactAccount.ProviderName,
			Product:    repo.ArtifactAccount.Product,
			Project:    repo.ArtifactAccount.Project,
		},
	}
	_, err := a.RevokeAuthroleRepoPolicy(ctx, req)
	return err
}
