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

package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/cache"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
)

type DeploymentRuntimeDeployer struct {
	// runtime is the runtime resource to be deployed.
	runtime *v1alpha1.DeploymentRuntime
	// components is the list of components used to deploy the runtime.
	components *component.ComponentList
	// snapshot is a snapshot of runtime-related resources.
	snapshot database.Snapshot
	// codeRepo is the code repo where the resource to be deployed is located.
	codeRepo v1alpha1.CodeRepo
	// repoProvider is the provider of the code repo.
	repoProvider v1alpha1.CodeRepoProvider
	// productID is the name of the runtime's product in nautes.
	productID string
	// productName is the name of the product that the runtime belongs to in the product provider
	productName string
	// usageController is a public cache controller for reading and writing public caches.
	usageController UsageController
	// cache is the last deployment cache.
	cache    DeploymentRuntimeSyncHistory
	newCache DeploymentRuntimeSyncHistory
}

// DeploymentRuntimeSyncHistory records the deployment information of the deployment runtime.
type DeploymentRuntimeSyncHistory struct {
	// Cluster is the name of the cluster where the runtime is deployed.
	Cluster string `json:"cluster,omitempty"`
	// Spaces is a list of spaces that have been used by the runtime.
	Spaces utils.StringSet `json:"spaces,omitempty"`
	// Account is the machine account name used by the runtime.
	Account *component.MachineAccount `json:"account,omitempty"`
	// Permissions records the access permissions of the Runtime machine account for how many key information items it possesses.
	Permissions []component.SecretInfo `json:"permissions,omitempty"`
}

func newDeploymentRuntimeDeployer(initInfo performerInitInfos) (taskPerformer, error) {
	deployRuntime := initInfo.runtime.(*v1alpha1.DeploymentRuntime)

	productID := deployRuntime.GetProduct()
	product, err := initInfo.NautesResourceSnapshot.GetProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("get product %s failed: %w", productID, err)
	}

	codeRepoName := deployRuntime.Spec.ManifestSource.CodeRepo
	codeRepo, err := initInfo.NautesResourceSnapshot.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, fmt.Errorf("get code repo %s failed: %w", codeRepoName, err)
	}

	provider, err := initInfo.NautesResourceSnapshot.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	history := &DeploymentRuntimeSyncHistory{}
	newHistory := &DeploymentRuntimeSyncHistory{}
	if initInfo.cache != nil {
		if err := json.Unmarshal(initInfo.cache.Raw, history); err != nil {
			return nil, fmt.Errorf("unmarshal history failed: %w", err)
		}
		_ = json.Unmarshal(initInfo.cache.Raw, newHistory)
	}
	if history.Spaces.Set == nil {
		history.Spaces = utils.NewStringSet()
		newHistory.Spaces = utils.NewStringSet()
	}

	usageController := UsageController{
		nautesNamespace: initInfo.NautesConfig.Nautes.Namespace,
		k8sClient:       initInfo.tenantK8sClient,
		clusterName:     initInfo.ClusterName,
	}

	return &DeploymentRuntimeDeployer{
		runtime:         deployRuntime,
		components:      initInfo.Components,
		snapshot:        initInfo.NautesResourceSnapshot,
		codeRepo:        *codeRepo,
		repoProvider:    *provider,
		productID:       productID,
		productName:     product.Spec.Name,
		usageController: usageController,
		cache:           *history,
		newCache:        *newHistory,
	}, nil
}

func (drd *DeploymentRuntimeDeployer) Deploy(ctx context.Context) (interface{}, error) {
	err := drd.deploy(ctx)
	return drd.newCache, err
}

// deploy will deploy the deployment runtime in the cluster.
// Main steps:
// 1.Create products in various components.
// 2.Create a space and machine account in the cluster.
// 3.Get the machine account information.
// 4.Create an account in the secret manager and update the account permissions.
// 5.Deploy the application.
func (drd *DeploymentRuntimeDeployer) deploy(ctx context.Context) error {
	productUsage, err := drd.usageController.GetProductUsage(ctx, drd.productID)
	if err != nil {
		return fmt.Errorf("get usage failed: %w", err)
	}
	if productUsage == nil {
		tmp := cache.NewEmptyProductUsage()
		productUsage = &tmp
	}
	defer func() {
		err := drd.usageController.UpdateProductUsage(context.TODO(), drd.productID, *productUsage)
		if err != nil {
			logger.Error(err, "update product usage failed")
		}
	}()

	productUsage.Runtimes.Insert(drd.runtime.Name)
	if err := CreateProduct(ctx, drd.productID, drd.productName, *drd.components); err != nil {
		return fmt.Errorf("create product in components failed: %w", err)
	}

	spaces := drd.runtime.GetNamespaces()
	for i := range spaces {
		if err := drd.components.MultiTenant.CreateSpace(ctx, drd.productID, spaces[i]); err != nil {
			return fmt.Errorf("create space failed: %w", err)
		}
	}
	productUsage.Space.UpdateSpaceUsage(drd.runtime.Name, utils.NewStringSet(spaces...))
	drd.newCache.Spaces.Insert(spaces...)

	accountName := drd.runtime.GetAccount()
	if drd.cache.Account != nil && drd.cache.Account.Name != accountName {
		oldAccountName := drd.cache.Account.Name
		err := revokeAccountPermissionsInSecretManagement(ctx, *drd.cache.Account, drd.cache.Permissions, *drd.components, drd.runtime.Name, productUsage.Account)
		if err != nil {
			return fmt.Errorf("revoke old account permissions failed: %w", err)
		}
		if err := drd.deleteSpacesUser(ctx, &productUsage.Account); err != nil {
			return fmt.Errorf("delete space user failed: %w", err)
		}

		if AccountIsRemovable(productUsage.Account, oldAccountName, drd.runtime.Name) {
			if err := DeleteProductAccount(ctx, *drd.cache.Account, *drd.components); err != nil {
				return fmt.Errorf("delete account failed: %w", err)
			}
		}
		productUsage.Account.DeleteRuntime(oldAccountName, drd.runtime.Name)
	}

	if err := drd.components.MultiTenant.CreateAccount(ctx, drd.productID, accountName); err != nil {
		return fmt.Errorf("create account in product failed: %w", err)
	}

	if err := drd.addSpacesUser(ctx, &productUsage.Account); err != nil {
		return err
	}

	account, err := drd.components.MultiTenant.GetAccount(ctx, drd.productID, accountName)
	if err != nil {
		return fmt.Errorf("get account %s's info failed: %w", accountName, err)
	}

	if _, err := drd.components.SecretManagement.CreateAccount(ctx, *account); err != nil {
		return fmt.Errorf("create account in secret management failed: %w", err)
	}

	newPermissions := drd.GetPermissionsFromRuntime()
	err = syncAccountPermissionsInSecretManagement(ctx, *account, newPermissions, drd.cache.Permissions, *drd.components, drd.runtime.Name, &productUsage.Account)
	if err != nil {
		return fmt.Errorf("sync permission in secret management failed: %w", err)
	}
	drd.newCache.Permissions = newPermissions
	drd.newCache.Account = account

	outdateSpaces := drd.cache.Spaces.Difference(productUsage.Space.ListSpaces().Set)
	for space := range outdateSpaces {
		if err := drd.components.MultiTenant.DeleteSpace(ctx, drd.productID, space); err != nil {
			return fmt.Errorf("delete outdate spaces failed: %w", err)
		}
		drd.newCache.Spaces.Delete(space)
	}

	app := drd.buildApp()
	return drd.components.Deployment.CreateApp(ctx, app)
}

func (drd *DeploymentRuntimeDeployer) Delete(ctx context.Context) (interface{}, error) {
	err := drd.delete(ctx)
	return &drd.newCache, err
}

// delete will clean up runtime-related resources deployed in the cluster.
// Main steps:
// 1.Get the machine account information.
// 2.Delete the app.
// 3.Clean up the account information in secret management.
// 4.Clean up invalid data in the product.
func (drd *DeploymentRuntimeDeployer) delete(ctx context.Context) error {
	productUsage, err := drd.usageController.GetProductUsage(ctx, drd.productID)
	if err != nil {
		return fmt.Errorf("get usage failed: %w", err)
	}
	if productUsage == nil {
		return nil
	}
	defer func() {
		err := drd.usageController.UpdateProductUsage(context.TODO(), drd.productID, *productUsage)
		if err != nil {
			logger.Error(err, "update product usage failed")
		}
	}()

	productUsage.Runtimes.Delete(drd.runtime.Name)
	productUsage.Space.DeleteRuntime(drd.runtime.Name)

	app := drd.buildApp()
	if err := drd.components.Deployment.DeleteApp(ctx, app); err != nil {
		return err
	}

	if drd.cache.Account != nil {
		err := revokeAccountPermissionsInSecretManagement(ctx, *drd.cache.Account, drd.cache.Permissions, *drd.components, drd.runtime.Name, productUsage.Account)
		if err != nil {
			return fmt.Errorf("revoke permission in secret management failed: %w", err)
		}

		if AccountIsRemovable(productUsage.Account, drd.cache.Account.Name, drd.runtime.Name) {
			if err := DeleteProductAccount(ctx, *drd.cache.Account, *drd.components); err != nil {
				return fmt.Errorf("delete product account failed: %w", err)
			}
		}
		productUsage.Account.DeleteRuntime(drd.cache.Account.Name, drd.runtime.Name)
	}

	productSpaces := productUsage.Space.ListSpaces(cache.ExcludedRuntimes([]string{drd.runtime.Name}))
	for _, space := range drd.cache.Spaces.Difference(productSpaces.Set).UnsortedList() {
		if err := drd.components.MultiTenant.DeleteSpace(ctx, drd.productID, space); err != nil {
			return fmt.Errorf("delete space failed: %w", err)
		}
	}

	if productUsage.Runtimes.Len() == 0 {
		if err := DeleteProduct(ctx, drd.productID, drd.productName, *drd.components); err != nil {
			return fmt.Errorf("delete product in components failed: %w", err)
		}
	}

	return nil
}

func (drd *DeploymentRuntimeDeployer) addSpacesUser(ctx context.Context, usage *cache.AccountUsage) error {
	spaces := drd.runtime.GetNamespaces()
	accountName := drd.runtime.GetAccount()
	for i := range spaces {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: drd.productID,
				Name:    spaces[i],
			},
			User:         accountName,
			RequestScope: component.RequestScopeAccount,
		}
		if err := drd.components.MultiTenant.AddSpaceUser(ctx, req); err != nil {
			return fmt.Errorf("add user %s to space %s failed: %w", req.User, req.Resource.Name, err)
		}
	}

	usage.UpdateSpaces(accountName, drd.runtime.Name, utils.NewStringSet(spaces...))
	return nil
}

func (drd *DeploymentRuntimeDeployer) deleteSpacesUser(ctx context.Context, usage *cache.AccountUsage) error {
	accountName := drd.cache.Account.Name
	accountSpaces := usage.Accounts[accountName].ListAccountSpaces(cache.ExcludedRuntimes([]string{drd.runtime.Name}))
	revokeSpaces := drd.cache.Spaces.Difference(accountSpaces.Set).UnsortedList()
	for _, space := range revokeSpaces {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: drd.productID,
				Name:    space,
			},
			User:         accountName,
			RequestScope: component.RequestScopeAccount,
		}

		if err := drd.components.MultiTenant.DeleteSpaceUser(ctx, req); err != nil {
			return fmt.Errorf("delete user %s from space %s failed: %w", req.User, req.Resource.Name, err)
		}
	}
	return nil
}

func (drd *DeploymentRuntimeDeployer) buildApp() component.Application {
	spaces := make([]component.Space, len(drd.runtime.GetNamespaces()))
	for i, ns := range drd.runtime.GetNamespaces() {
		spaces[i] = component.Space{
			ResourceMetaData: component.ResourceMetaData{
				Product: drd.productID,
				Name:    ns,
			},
			SpaceType: component.SpaceTypeKubernetes,
			Kubernetes: &component.SpaceKubernetes{
				Namespace: ns,
			},
		}
	}

	app := component.Application{
		ResourceMetaData: component.ResourceMetaData{
			Product: drd.productID,
			Name:    drd.runtime.Name,
		},
		Git: &component.ApplicationGit{
			URL:      drd.codeRepo.Spec.URL,
			Revision: drd.runtime.Spec.ManifestSource.TargetRevision,
			Path:     drd.runtime.Spec.ManifestSource.Path,
			CodeRepo: drd.codeRepo.Name,
		},
		Destinations: spaces,
	}
	return app
}

func (drd *DeploymentRuntimeDeployer) GetPermissionsFromRuntime() []component.SecretInfo {
	var permission []component.SecretInfo

	permission = append(permission, component.SecretInfo{
		Type: component.SecretTypeCodeRepo,
		CodeRepo: &component.CodeRepo{
			ProviderType: drd.repoProvider.Spec.ProviderType,
			ID:           drd.codeRepo.Name,
			User:         "default",
			Permission:   component.CodeRepoPermissionReadOnly,
		},
	})

	return permission
}
