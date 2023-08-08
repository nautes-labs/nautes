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

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
)

const (
	ROLE_NAME_KEY_ARGO    = "Argo"
	ROLE_NAME_KEY_RUNTIME = "Runtime"
	AUTH_PATH_FORMAT      = "auth/%s/config"
	SECRET_PATH_FORMAT    = "kubernetes/%s/default/admin"
)

type SyncResult struct {
	Secret *SyncSecretResult
	Auth   *SyncAuthResult
	Error  error
}

type SyncSecretResult struct {
	SecretVersion int
}

type SyncAuthResult struct {
}

func (vc *VaultClient) Delete(ctx context.Context, cluster *nautescrd.Cluster) error {
	ctx = context.WithValue(ctx, CONTEXT_KEY_CFG, *vc.Configs)
	defer vc.Logout()

	err := vc.deleteCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("delete cluster failed: %w", err)
	}
	return nil
}

func (vc *VaultClient) Sync(ctx context.Context, cluster, lastCluster *nautescrd.Cluster) (*secretclient.SyncResult, error) {
	ctx = context.WithValue(ctx, CONTEXT_KEY_CFG, *vc.Configs)
	defer vc.Logout()

	if cluster.Name == vc.TenantAuthName {
		return nil, fmt.Errorf("tenant cluster can not be modified")
	}

	result, err := vc.SyncCluster(ctx, cluster, lastCluster)
	if err != nil {
		vc.CleanCluster(ctx, lastCluster, cluster, result)
		return nil, fmt.Errorf("sync cluster failed: %w", err)
	}

	return &secretclient.SyncResult{
		SecretID: fmt.Sprint(result.Secret.SecretVersion),
	}, nil
}

func (vc *VaultClient) SyncCluster(ctx context.Context, cluster, lastCluster *nautescrd.Cluster) (*SyncResult, error) {
	result := &SyncResult{}

	secretResult, err := vc.SyncSecret(cluster, lastCluster)
	result.Secret = secretResult
	if err != nil {
		return result, fmt.Errorf("sync secret failed: %w", err)
	}

	if result.Secret.SecretVersion != 0 {
		authResult, err := vc.syncAuth(ctx, cluster, lastCluster)
		result.Auth = authResult
		if err != nil {
			return result, fmt.Errorf("sync auth failed: %w", err)
		}
	}

	return result, nil
}

func (vc *VaultClient) SyncSecret(cluster, lastCluster *nautescrd.Cluster) (*SyncSecretResult, error) {
	var deleteCluster, newCluster *nautescrd.Cluster
	noSecretVerion := false
	result := &SyncSecretResult{
		SecretVersion: 0,
	}

	// Check cluster type is a host cluster, if true ,check it is change from host cluster
	if cluster.Spec.ClusterType != nautescrd.CLUSTER_TYPE_PHYSICAL {
		newCluster = cluster
	} else if lastCluster != nil && lastCluster.Spec.ClusterType == nautescrd.CLUSTER_TYPE_VIRTUAL {
		deleteCluster = lastCluster
		newCluster = nil
		noSecretVerion = true
	}

	if err := vc.createSecret(newCluster); err != nil {
		return result, fmt.Errorf("create secret failed: %w", err)
	}

	if err := vc.deleteSecret(deleteCluster); err != nil {
		return result, fmt.Errorf("delete secret failed: %w", err)
	}

	if noSecretVerion {
		return result, nil
	}

	secretMeta, err := vc.getSecretMeta(cluster)
	if err != nil {
		return result, fmt.Errorf("secret not found in vault")
	}

	result.SecretVersion = secretMeta.CurrentVersion

	return result, nil
}

func (vc *VaultClient) syncAuth(ctx context.Context, cluster, lastCluster *nautescrd.Cluster) (*SyncAuthResult, error) {
	result := &SyncAuthResult{}

	switch usage := cluster.Spec.Usage; usage {
	case nautescrd.CLUSTER_USAGE_WORKER:
		if err := vc.createOrUpdateAuth(ctx, cluster); err != nil {
			return result, err
		}
	case nautescrd.CLUSTER_USAGE_HOST:
		if err := vc.deleteAuth(ctx, lastCluster); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (vc *VaultClient) createOrUpdateAuth(ctx context.Context, cluster *nautescrd.Cluster) error {
	if err := vc.createAuth(cluster); err != nil {
		return fmt.Errorf("create auth failed: %w", err)
	}

	if err := vc.createRole(cluster); err != nil {
		return fmt.Errorf("create role failed: %w", err)
	}

	if err := vc.grantTenantPermission(ctx, cluster, cluster.Name); err != nil {
		return fmt.Errorf("grant tenant code repo permission failed, %w", err)
	}

	if err := vc.grantClusterPermission(ctx, cluster, vc.TenantAuthName); err != nil {
		return fmt.Errorf("grant cluster permission failed: %w", err)
	}

	return nil
}

func (vc *VaultClient) deleteAuth(ctx context.Context, cluster *nautescrd.Cluster) error {
	if err := vc.disableAuth(cluster); err != nil {
		return fmt.Errorf("disable auth failed: %w", err)
	}

	if err := vc.revokeClusterPermission(ctx, cluster, vc.TenantAuthName); err != nil {
		return fmt.Errorf("revoke clsuter permission failed: %w", err)
	}

	return nil
}

func (vc *VaultClient) deleteCluster(ctx context.Context, lastCluster *nautescrd.Cluster) error {
	if err := vc.deleteSecret(lastCluster); err != nil {
		return fmt.Errorf("delete secret failed: %w", err)
	}

	if err := vc.deleteAuth(ctx, lastCluster); err != nil {
		return fmt.Errorf("delete auth failed: %w", err)
	}

	return nil
}

func (vc *VaultClient) CleanCluster(ctx context.Context, cluster, lastCluster *nautescrd.Cluster, result *SyncResult) error {
	errors := []error{}

	if isNewSecret(cluster, lastCluster) {
		vc.log.V(1).Info("clean up cluster secret")
		if err := vc.deleteSecret(cluster); err != nil {
			errors = append(errors, err)
		}
	}

	if isNewAuth(cluster, lastCluster) {
		vc.log.V(1).Info("clean up cluster auth")
		if err := vc.disableAuth(cluster); err != nil {
			errors = append(errors, err)
		}

		if err := vc.revokeClusterPermission(ctx, cluster, vc.TenantAuthName); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("clean up cluster failed: %v", errors)
	}
	return nil
}

func isNewSecret(cluster, lastCluster *nautescrd.Cluster) bool {
	if cluster == nil {
		return false
	}

	if lastCluster == nil {
		return true
	}

	if cluster.Spec.ClusterType == nautescrd.CLUSTER_TYPE_VIRTUAL &&
		cluster.Spec.ClusterType != lastCluster.Spec.ClusterType {
		return true
	}

	return false
}

func isNewAuth(cluster, lastCluster *nautescrd.Cluster) bool {
	if cluster == nil {
		return false
	}

	if lastCluster == nil {
		return true
	}

	if cluster.Spec.Usage == nautescrd.CLUSTER_USAGE_WORKER &&
		cluster.Spec.Usage != lastCluster.Spec.Usage {
		return true
	}
	return false
}
