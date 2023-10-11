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

package syncer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

type DeploymentRuntimeDeployer struct {
	runtime         *v1alpha1.DeploymentRuntime
	deployer        Deployment
	productMgr      MultiTenant
	secMgr          SecretManagement
	db              database.Database
	codeRepo        v1alpha1.CodeRepo
	repoProvider    v1alpha1.CodeRepoProvider
	productID       string
	productName     string
	usageController UsageController
	rawCache        *runtime.RawExtension
	cache           DeploymentRuntimeSyncHistory
	newCache        DeploymentRuntimeSyncHistory
}

type DeploymentRuntimeSyncHistory struct {
	Cluster  string    `json:"cluster,omitempty"`
	Spaces   StringSet `json:"spaces,omitempty"`
	CodeRepo string    `json:"codeRepo,omitempty"`
}

func newDeploymentRuntimeDeployer(initInfo runnerInitInfos) (taskRunner, error) {
	multiTenant := initInfo.Components.MultiTenant
	deployer := initInfo.Components.Deployment
	secMgr := initInfo.Components.SecretManagement
	deployRuntime := initInfo.runtime.(*v1alpha1.DeploymentRuntime)

	productID := deployRuntime.GetProduct()
	product, err := initInfo.NautesDB.GetProduct(productID)
	if err != nil {
		return nil, fmt.Errorf("get product %s failed: %w", productID, err)
	}

	codeRepoName := deployRuntime.Spec.ManifestSource.CodeRepo
	codeRepo, err := initInfo.NautesDB.GetCodeRepo(codeRepoName)
	if err != nil {
		return nil, fmt.Errorf("get code repo %s failed: %w", codeRepoName, err)
	}

	provider, err := initInfo.NautesDB.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	history := &DeploymentRuntimeSyncHistory{}
	if initInfo.cache != nil {
		if err := json.Unmarshal(initInfo.cache.Raw, history); err != nil {
			return nil, fmt.Errorf("unmarshal history failed: %w", err)
		}
	}
	if history.Spaces.Set == nil {
		history.Spaces = StringSet{
			Set: sets.New[string](),
		}
	}

	usageController := UsageController{
		nautesNamespace: initInfo.NautesConfig.Nautes.Namespace,
		k8sClient:       initInfo.tenantK8sClient,
		clusterName:     initInfo.ClusterName,
		runtime:         initInfo.runtime,
		productID:       productID,
	}

	return &DeploymentRuntimeDeployer{
		runtime:         deployRuntime,
		deployer:        deployer,
		productMgr:      multiTenant,
		secMgr:          secMgr,
		db:              initInfo.NautesDB,
		codeRepo:        *codeRepo,
		repoProvider:    *provider,
		productID:       productID,
		productName:     product.Spec.Name,
		usageController: usageController,
		rawCache:        initInfo.cache,
		cache:           *history,
		newCache:        *history,
	}, nil
}

func (drd *DeploymentRuntimeDeployer) Deploy(ctx context.Context) (*runtime.RawExtension, error) {
	err := drd.deploy(ctx)

	cache, convertErr := json.Marshal(drd.newCache)
	if convertErr != nil {
		return drd.rawCache, convertErr
	}

	return &runtime.RawExtension{
		Raw: cache,
	}, err
}

func (drd *DeploymentRuntimeDeployer) deploy(ctx context.Context) error {
	usage, err := drd.usageController.AddProductUsage(ctx)
	if err != nil {
		return fmt.Errorf("add product usage failed: %w", err)
	}

	if err := drd.initEnvironment(ctx, *usage); err != nil {
		return err
	}

	userName := drd.runtime.GetName()
	user, err := drd.productMgr.GetUser(ctx, drd.productID, userName)
	if err != nil {
		return fmt.Errorf("get user %s's info failed: %w", userName, err)
	}

	if err := drd.syncUserInSecretDatabase(ctx, *user); err != nil {
		return err
	}

	return drd.deployApp(ctx)
}

func (drd *DeploymentRuntimeDeployer) Delete(ctx context.Context) (*runtime.RawExtension, error) {
	err := drd.delete(ctx)
	cache, convertErr := json.Marshal(drd.newCache)
	if convertErr != nil {
		return drd.rawCache, convertErr
	}

	return &runtime.RawExtension{
		Raw: cache,
	}, err
}

func (drd *DeploymentRuntimeDeployer) delete(ctx context.Context) error {
	usage, err := drd.usageController.GetProductUsage(ctx)
	if err != nil {
		return err
	}
	if usage == nil {
		return nil
	}

	usage.Runtimes.Delete(drd.runtime.Name)

	user, err := drd.productMgr.GetUser(ctx, drd.productID, drd.runtime.GetName())
	if err != nil {
		if !IsUserNotFound(err) {
			return fmt.Errorf("get user %s's info failed: %w", drd.runtime.GetName(), err)
		}
	}

	if err := drd.deleteDeploymentApps(ctx, *usage); err != nil {
		return err
	}

	if user != nil {
		if err := drd.deleteUserInSecretDatabase(ctx, *user); err != nil {
			return err
		}
	}

	if err := drd.cleanUpProduct(ctx, user, *usage); err != nil {
		return err
	}
	return drd.usageController.DeleteProductUsage(ctx)
}

func (drd *DeploymentRuntimeDeployer) deleteUserInSecretDatabase(ctx context.Context, user User) error {
	if drd.cache.CodeRepo != "" {
		if err := drd.secMgr.RevokePermission(ctx, buildSecretInfoCodeRepo(drd.repoProvider.Spec.ProviderType, drd.cache.CodeRepo), user); err != nil {
			return fmt.Errorf("revoke code repo %s readonly permission from user %s failed: %w", drd.cache.CodeRepo, user.Name, err)
		}
		drd.newCache.CodeRepo = ""
	}

	if err := drd.secMgr.DeleteUser(ctx, user); err != nil {
		return fmt.Errorf("delete user %s in secret database failed: %w", user.Name, err)
	}
	return nil
}

func (drd *DeploymentRuntimeDeployer) deleteDeploymentApps(ctx context.Context, usage ProductUsage) error {
	app := drd.buildApp()
	if err := drd.deployer.DeleteApp(ctx, app); err != nil {
		return err
	}

	if usage.Runtimes.Len() == 0 {
		err := drd.deployer.DeleteProductUser(ctx, PermissionRequest{
			RequestScope: RequestScopeProduct,
			Resource: Resource{
				Product: drd.productID,
				Name:    drd.productID,
			},
			User:       drd.productName,
			Permission: Permission{},
		})
		if err != nil {
			return fmt.Errorf("revoke permission from deployment product failed: %w", err)
		}

		err = drd.deployer.DeleteProduct(ctx, drd.productID)
		if err != nil {
			return fmt.Errorf("delete deployment product failed: %w", err)
		}
	}
	return nil
}

func (drd *DeploymentRuntimeDeployer) cleanUpProduct(ctx context.Context, user *User, usage ProductUsage) error {
	if user != nil {
		if err := drd.productMgr.DeleteUser(ctx, drd.productID, user.Name); err != nil {
			return fmt.Errorf("delete user %s in product %s failed: %w", user.Name, drd.productID, err)
		}
	}

	spacesInUsed := drd.getSpaceUsage(usage)
	for _, ns := range drd.cache.Spaces.UnsortedList() {
		if spacesInUsed.Has(ns) {
			continue
		}
		err := drd.productMgr.DeleteSpace(ctx, drd.productID, ns)
		if err != nil {
			return fmt.Errorf("delete space %s failed: %w", ns, err)
		}
		drd.newCache.Spaces.Delete(ns)
	}

	if usage.Runtimes.Len() == 0 {
		if err := drd.productMgr.DeleteProduct(ctx, drd.productID); err != nil {
			return fmt.Errorf("delete product %s failed: %w", drd.productID, err)
		}
	}
	return nil
}

func (drd *DeploymentRuntimeDeployer) getSpaceUsage(usage ProductUsage) sets.Set[string] {
	spaces := sets.New[string]()
	for _, dr := range usage.Runtimes.UnsortedList() {
		dr, err := drd.db.GetRuntime(dr, v1alpha1.RuntimeTypeDeploymentRuntime)
		if err != nil {
			continue
		}
		spaces.Insert(dr.GetNamespaces()...)
	}
	return spaces
}

func (drd *DeploymentRuntimeDeployer) deployApp(ctx context.Context) error {
	if err := drd.deployer.CreateProduct(ctx, drd.productID); err != nil {
		return fmt.Errorf("create deployment product failed: %w", err)
	}

	if err := drd.deployer.AddProductUser(ctx, PermissionRequest{
		RequestScope: RequestScopeProduct,
		Resource: Resource{
			Product: drd.productID,
			Name:    drd.productID,
		},
		User:       drd.productName,
		Permission: Permission{},
	}); err != nil {
		return fmt.Errorf("grant deployment product %s admin permission to group %s failed: %w",
			drd.productID, drd.productName, err)
	}

	app := drd.buildApp()
	return drd.deployer.CreateApp(ctx, app)
}

func (drd *DeploymentRuntimeDeployer) syncUserInSecretDatabase(ctx context.Context, user User) error {
	if err := drd.secMgr.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user %s failed: %w", user.Name, err)
	}

	err := drd.secMgr.GrantPermission(ctx, buildSecretInfoCodeRepo(drd.repoProvider.Spec.ProviderType, drd.codeRepo.Name), user)
	if err != nil {
		return fmt.Errorf("grant coderepo readonly permission to user %s failed: %w", user.Name, err)
	}

	if drd.codeRepo.Name != drd.cache.CodeRepo && drd.cache.CodeRepo != "" {
		err := drd.secMgr.RevokePermission(ctx, buildSecretInfoCodeRepo(drd.repoProvider.Spec.ProviderType, drd.cache.CodeRepo), user)
		if err != nil {
			return fmt.Errorf("revoke coderepo readonly permission from user %s failed: %w", user.Name, err)
		}
	}

	drd.newCache.CodeRepo = drd.codeRepo.Name
	return nil
}

func (drd *DeploymentRuntimeDeployer) initEnvironment(ctx context.Context, usage ProductUsage) error {
	if err := drd.productMgr.CreateProduct(ctx, drd.productID); err != nil {
		return fmt.Errorf("create product failed: %w", err)
	}

	userName := drd.runtime.GetName()
	if err := drd.productMgr.CreateUser(ctx, drd.productID, userName); err != nil {
		return fmt.Errorf("create user failed: %w", err)
	}

	for _, ns := range drd.runtime.GetNamespaces() {
		err := drd.productMgr.CreateSpace(ctx, drd.productID, ns)
		if err != nil {
			return fmt.Errorf("create space %s failed: %w", ns, err)
		}
		drd.newCache.Spaces.Insert(ns)

		if err := drd.productMgr.AddSpaceUser(ctx, PermissionRequest{
			RequestScope: RequestScopeUser,
			Resource: Resource{
				Product: drd.productID,
				Name:    ns,
			},
			User:       userName,
			Permission: Permission{},
		}); err != nil {
			return fmt.Errorf("add user %s to space %s failed: %w", userName, ns, err)
		}
	}

	spacesInUsed := drd.getSpaceUsage(usage)
	for _, ns := range drd.getUnusedSpacesInCache() {
		if spacesInUsed.Has(ns) {
			drd.newCache.Spaces.Delete(ns)
			continue
		}
		err := drd.productMgr.DeleteSpace(ctx, drd.productID, ns)
		if err != nil {
			return fmt.Errorf("delete space %s failed: %w", ns, err)
		}
		drd.newCache.Spaces.Delete(ns)
	}

	return nil
}

func (drd *DeploymentRuntimeDeployer) getUnusedSpacesInCache() []string {
	newSpaces := sets.New(drd.runtime.GetNamespaces()...)
	oldSpaces := drd.cache.Spaces
	return oldSpaces.Difference(newSpaces).UnsortedList()
}

func (drd *DeploymentRuntimeDeployer) buildApp() Application {
	spaces := make([]Space, len(drd.runtime.GetNamespaces()))
	for i, ns := range drd.runtime.GetNamespaces() {
		spaces[i] = Space{
			Resource: Resource{
				Product: drd.productID,
				Name:    ns,
			},
			SpaceType: SpaceTypeKubernetes,
			Kubernetes: SpaceKubernetes{
				Namespace: ns,
			},
		}
	}

	app := Application{
		Resource: Resource{
			Product: drd.productID,
			Name:    drd.runtime.Name,
		},
		Git: &ApplicationGit{
			URL:      drd.codeRepo.Spec.URL,
			Revision: drd.runtime.Spec.ManifestSource.TargetRevision,
			Path:     drd.runtime.Spec.ManifestSource.Path,
			CodeRepo: drd.codeRepo.Name,
		},
		Destinations: spaces,
	}
	return app
}

func buildSecretInfoCodeRepo(providerType, name string) SecretInfo {
	return SecretInfo{
		Type: SecretTypeCodeRepo,
		CodeRepo: &CodeRepo{
			ProviderType: providerType,
			ID:           name,
			User:         "default",
			Permission:   CodeRepoPermissionReadOnly,
		},
	}
}
