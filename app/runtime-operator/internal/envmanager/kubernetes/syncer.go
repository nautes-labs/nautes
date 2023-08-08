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

package kubernetes

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"github.com/nautes-labs/nautes/pkg/kubeconvert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
)

type Syncer struct {
	client.Client
}

// GetAccessInfo get connect info from cluster resource
func (m Syncer) GetAccessInfo(ctx context.Context, cluster nautescrd.Cluster) (*interfaces.AccessInfo, error) {
	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return nil, fmt.Errorf("get secret client from context failed")
	}

	accessInfo, err := secClient.GetAccessInfo(ctx, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	restConfig, err := kubeconvert.ConvertStringToRestConfig(accessInfo)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	return &interfaces.AccessInfo{
		Name:       cluster.Name,
		Type:       interfaces.ACCESS_TYPE_K8S,
		Kubernetes: restConfig,
	}, nil
}

// Sync create or update a usable env for the next step, it will create namespaces, rolebinding and other resources runtime required.
func (m Syncer) Sync(ctx context.Context, task interfaces.RuntimeSyncTask) (*interfaces.EnvironmentDeploymentResult, error) {
	destCluster, err := newDestCluster(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("create dest cluster client failed: %w", err)
	}

	productCodeRepo, err := getProductCodeRepo(ctx, m.Client, task.Product.Name, task.NautesCfg.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get product coderepo failed: %w", err)
	}
	if err := destCluster.syncProductNamespace(ctx, productCodeRepo); err != nil {
		return nil, fmt.Errorf("sync product namespace failed: %w", err)
	}

	if err := destCluster.syncProductAuthority(ctx); err != nil {
		return nil, fmt.Errorf("sync product authority failed: %w", err)
	}

	if err := destCluster.syncRuntimeNamespace(ctx); err != nil {
		return nil, fmt.Errorf("sync runtime namespace failed: %w", err)
	}

	if err := destCluster.syncRelationShip(ctx); err != nil {
		return nil, fmt.Errorf("sync relationship namespace failed: %w", err)
	}

	if err := destCluster.SyncRole(ctx); err != nil {
		return nil, fmt.Errorf("sync role %s failed: %w", destCluster.Runtime.GetName(), err)
	}

	syncResult := &interfaces.EnvironmentDeploymentResult{}
	repos, err := m.getRepos(ctx, task)
	if err != nil {
		syncResult.Error = err
	}

	err = destCluster.SyncRepo(ctx, repos)
	if err != nil {
		syncResult.Error = err
	}

	namespaces, err := utils.GetProductNamespacesInCluster(ctx, m.Client, task.Product.Name, task.Cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("get namespaces failed: %w", err)
	}

	notUsedNamespace, err := destCluster.GetNotUsedNamespaces(ctx, namespaces)
	for _, namespace := range notUsedNamespace {
		if err := destCluster.deleteNamespace(ctx, namespace); err != nil {
			return nil, fmt.Errorf("delete runtime namespace failed: %w", err)
		}
	}

	return syncResult, nil
}

// Remove will cleaa up resouces Sync create.
func (m Syncer) Remove(ctx context.Context, task interfaces.RuntimeSyncTask) error {
	logger := log.FromContext(ctx)

	destCluster, err := newDestCluster(ctx, task)
	if err != nil {
		return fmt.Errorf("create dest cluster client failed: %w", err)
	}

	if err := destCluster.DeleteRole(ctx); err != nil {
		return fmt.Errorf("delete secret role failed: %w", err)
	}

	namespaces, err := utils.GetProductNamespacesInCluster(ctx, m.Client, task.Product.Name, task.Cluster.Name)
	if err != nil {
		return fmt.Errorf("get namespaces failed: %w", err)
	}

	notUsedNamespace, err := destCluster.GetNotUsedNamespaces(ctx, namespaces)
	for _, namespace := range notUsedNamespace {
		if err := destCluster.deleteNamespace(ctx, namespace); err != nil {
			return fmt.Errorf("delete runtime namespace failed: %w", err)
		}
	}
	if len(namespaces) == 1 {
		logger.V(1).Info("runtime under product namespace is zero, it will be delete", "NamespaceName", task.Product.Name)

		productCodeRepo, err := getProductCodeRepo(ctx, m.Client, task.Product.Name, task.NautesCfg.Nautes.Namespace)
		if err != nil {
			return fmt.Errorf("get product coderepo failed: %w", err)
		}

		if err := destCluster.deleteProductNamespace(ctx, productCodeRepo); err != nil {
			return fmt.Errorf("delete product namespace failed: %w", err)
		}
	}

	return nil
}

func (m Syncer) getRepos(ctx context.Context, task interfaces.RuntimeSyncTask) ([]interfaces.SecretInfo, error) {
	artifactRepos := &nautescrd.ArtifactRepoList{}
	listOpts := []client.ListOption{
		client.InNamespace(task.Product.Name),
	}
	if err := m.Client.List(ctx, artifactRepos, listOpts...); err != nil {
		return nil, fmt.Errorf("get repo list failed: %w", err)
	}

	repos := []interfaces.SecretInfo{}
	for _, artifactRepo := range artifactRepos.Items {
		provider := &nautescrd.ArtifactRepoProvider{}
		key := types.NamespacedName{
			Namespace: task.NautesCfg.Nautes.Namespace,
			Name:      artifactRepo.Spec.ArtifactRepoProvider,
		}
		if err := m.Get(ctx, key, provider); err != nil {
			return nil, fmt.Errorf("get artifact provider failed: %w", err)
		}

		repos = append(repos, interfaces.SecretInfo{
			Type: interfaces.SECRET_TYPE_ARTIFACT,
			AritifaceRepo: &interfaces.ArifactRepo{
				ProviderName: provider.Name,
				RepoType:     provider.Spec.ProviderType,
				ID:           artifactRepo.Name,
				User:         "default",
				Permission:   "readonly",
			},
		})
	}

	return repos, nil
}

func getProductCodeRepo(ctx context.Context, k8sClient client.Client, productName, nautesNamespace string) (*nautescrd.CodeRepo, error) {
	coderepoList := &nautescrd.CodeRepoList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{nautescrd.LABEL_FROM_PRODUCT: productName}),
		client.InNamespace(nautesNamespace),
	}
	if err := k8sClient.List(ctx, coderepoList, listOpts...); err != nil {
		return nil, fmt.Errorf("get product %s code repo failed: %w", productName, err)
	}
	if len(coderepoList.Items) != 1 {
		return nil, fmt.Errorf("product %s code repo is not unique", productName)
	}
	productCodeRepo := coderepoList.Items[0]

	return &productCodeRepo, nil
}
