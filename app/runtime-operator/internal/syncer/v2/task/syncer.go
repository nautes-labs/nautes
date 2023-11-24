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
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	kubeconvert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/cache"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	// NewFunctionMapDeployment stores the new method of the deployment component.
	NewFunctionMapDeployment = map[string]component.NewDeployment{}
	// NewFunctionMapMultiTenant stores the new method of the multi tenant component.
	NewFunctionMapMultiTenant = map[string]component.NewMultiTenant{}
	// NewFunctionMapSecretManagement stores the new method of the secret management component.
	NewFunctionMapSecretManagement = map[string]component.NewSecretManagement{}
	// NewFunctionMapSecretSync stores the new method of the secret sync component.
	NewFunctionMapSecretSync = map[string]component.NewSecretSync{}
	// NewFunctionMapGateway stores the new method of the gateway component.
	NewFunctionMapGateway = map[string]component.NewGateway{}
	// NewFunctionMapEventListener stores the new method of the event listener component.
	NewFunctionMapEventListener = map[string]component.NewEventListener{}
	// ComponentFactoryMapPipeline stores the object of pipeline factory.
	ComponentFactoryMapPipeline = map[string]component.PipelineFactory{}
)

var (
	// newFunctionMapTaskPerformer records a set of New methods for TaskPerformer.
	// The index is a runtime type.
	newFunctionMapTaskPerformer = map[v1alpha1.RuntimeType]NewTaskPerformer{
		v1alpha1.RuntimeTypeDeploymentRuntime: newDeploymentRuntimeDeployer,
		v1alpha1.RuntimeTypePipelineRuntime:   newPipelineRuntimeDeployer,
	}
	// NewSnapshot is the method for generating a snapshot of the nautes resource.
	NewSnapshot = database.NewRuntimeSnapshot
)

var (
	logger = logf.Log.WithName("task")
)

// NewTaskPerformer will generate a taskPerformer instance.
type NewTaskPerformer func(initInfo performerInitInfos) (taskPerformer, error)

// performerInitInfos stores the information needed to initialize taskPerformer.
type performerInitInfos struct {
	*component.ComponentInitInfo
	// runtime is the runtime resource to be deployed.
	runtime v1alpha1.Runtime
	// cache is the information last deployed on the runtime.
	cache *pkgruntime.RawExtension
	// tenantK8sClient is the k8s client of tenant cluster.
	tenantK8sClient client.Client
}

// taskPerformer can deploy or clean up specific types of runtime.
type taskPerformer interface {
	// Deploy will deploy the runtime in the cluster and return the information to be cached in the runtime resource.
	//
	// Additional notes:
	// - The environment information and runtime information to be deployed have been passed in the new method of taskPerformer.
	//
	// Return:
	// - The cache will be written to the runtime regardless of success. If nil is returned, all cache in the runtime will be cleared.
	Deploy(ctx context.Context) (interface{}, error)
	// Delete will clean up the runtime deployed in the cluster.
	//
	// Additional notes:
	// - The environment information and runtime information to be deployed have been passed in the new method of taskPerformer.
	//
	// Return:
	// - If the cleanup is successful, the returned cache should be nil.  If the cleanup fails, the remaining resource information should be returned.
	Delete(ctx context.Context) (interface{}, error)
}

// Syncer can create tasks based on environment information and runtime.
type Syncer struct {
	KubernetesClient client.Client
	PluginMgr        component.PipelinePluginManager
}

// NewTask loads nautes resources from tenant cluster and initializes components.
// It will do the following things:
// 1.Get NautesConfig.
// 2.Get all the resources related to the product.
// 3.Get the cluster corresponding to the runtime.
// 4.Get the access method of the cluster.
// 5.Initialize all related components.
// 6.Create a taskPerformer based on the runtime type.
// 7.Return the task.
func (s *Syncer) NewTask(ctx context.Context, runtime v1alpha1.Runtime, componentsStatus *pkgruntime.RawExtension) (*Task, error) {
	cfg, err := configs.NewNautesConfigFromFile()
	if err != nil {
		return nil, fmt.Errorf("load nautes config failed: %w", err)
	}

	productName := runtime.GetProduct()
	snapshot, err := NewSnapshot(ctx, s.KubernetesClient, productName, cfg.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("get nautes snapshot failed: %w", err)
	}

	cluster, err := snapshot.GetClusterByRuntime(runtime)
	if err != nil {
		return nil, fmt.Errorf("get cluster by runtime failed: %w", err)
	}

	var hostCluster *v1alpha1.Cluster
	if cluster.Spec.ClusterType == v1alpha1.CLUSTER_TYPE_VIRTUAL {
		cluster, err := snapshot.GetCluster(cluster.Spec.HostCluster)
		if err != nil {
			return nil, fmt.Errorf("get host cluster failed: %w", err)
		}
		hostCluster = cluster
	}

	initInfo := &component.ComponentInitInfo{
		ClusterConnectInfo:     component.ClusterConnectInfo{},
		ClusterName:            cluster.GetName(),
		NautesResourceSnapshot: snapshot,
		RuntimeName:            runtime.GetName(),
		NautesConfig:           *cfg,
		Components:             &component.ComponentList{Deployment: nil, MultiTenant: nil, SecretManagement: nil, SecretSync: nil, EventListener: nil},
		PipelinePluginManager:  s.PluginMgr,
	}

	newSecManagement, ok := NewFunctionMapSecretManagement[string(cfg.Secret.RepoType)]
	if !ok {
		return nil, fmt.Errorf("unknown secret management type %s", cfg.Secret.RepoType)
	}

	secMgr, err := newSecManagement(v1alpha1.Component{}, initInfo)
	if err != nil {
		return nil, fmt.Errorf("create secret management failed: %w", err)
	}

	clusterConnectInfo, err := getClusterConnectInfo(ctx, secMgr)
	if err != nil {
		return nil, err
	}

	initInfo.ClusterConnectInfo = *clusterConnectInfo

	statuses := NewComponentStatus()
	if err := initComponents(initInfo, cluster, hostCluster, secMgr, statuses); err != nil {
		return nil, fmt.Errorf("init componentList failed: %w", err)
	}

	performerInitInfos := performerInitInfos{
		ComponentInitInfo: initInfo,
		runtime:           runtime,
		cache:             componentsStatus,
		tenantK8sClient:   s.KubernetesClient,
	}

	performer, err := newFunctionMapTaskPerformer[runtime.GetRuntimeType()](performerInitInfos)
	if err != nil {
		return nil, err
	}

	return &Task{
		runtimeCache:    componentsStatus,
		components:      initInfo.Components,
		performer:       performer,
		clusterName:     cluster.Name,
		nautesNamespace: cfg.Nautes.Namespace,
		tenantK8sClient: s.KubernetesClient,
		componentCache:  statuses,
	}, nil
}

const (
	ConfigMapNameClustersUsageCache = "clusters-usage-cache"
)

// Task is the definition of runtime deployment tasks, which contains the objects required for deploying the runtime.
type Task struct {
	// runtimeCache is the cache from the last deployment.
	runtimeCache *pkgruntime.RawExtension
	// components stores the component instances required for deployment tasks.
	components *component.ComponentList
	// performer stores the execution instance of the runtime deployment.
	performer taskPerformer
	// clusterName is the name of the cluster deployed by the runtime.
	clusterName string
	// nautesNamespace is the namespace where the nautes component is located
	nautesNamespace string
	// tenantK8sClient is the client of the tenant cluster
	tenantK8sClient client.Client
	// componentCache is the cache of components.
	componentCache *ComponentsStatus
}

type ComponentsStatus struct {
	status map[component.ComponentType]interface{}
}

func NewComponentStatus() *ComponentsStatus {
	statuses := &ComponentsStatus{
		status: map[component.ComponentType]interface{}{},
	}
	return statuses
}

func (t *Task) Run(ctx context.Context) (*pkgruntime.RawExtension, error) {
	defer t.CleanUp()
	defer t.UpdateComponentCache()

	status, err := t.performer.Deploy(ctx)
	runtimeCache, convertErr := json.Marshal(status)
	if convertErr != nil {
		return t.runtimeCache, convertErr
	}

	return &pkgruntime.RawExtension{
		Raw: runtimeCache,
	}, err
}

func (t *Task) Delete(ctx context.Context) (*pkgruntime.RawExtension, error) {
	defer t.CleanUp()
	defer t.UpdateComponentCache()

	status, err := t.performer.Delete(ctx)
	runtimeCache, convertErr := json.Marshal(status)
	if convertErr != nil {
		return t.runtimeCache, convertErr
	}
	return &pkgruntime.RawExtension{
		Raw: runtimeCache,
	}, err
}

// CleanUp will call the CleanUp method of all components to complete the task cleanup.
func (t *Task) CleanUp() {
	if t.components.Deployment != nil {
		err := t.components.Deployment.CleanUp()
		if err != nil {
			logger.Error(err, "")
		}
	}
	if t.components.MultiTenant != nil {
		err := t.components.MultiTenant.CleanUp()
		if err != nil {
			logger.Error(err, "")
		}
	}
	if t.components.SecretManagement != nil {
		err := t.components.SecretManagement.CleanUp()
		if err != nil {
			logger.Error(err, "")
		}
	}
	if t.components.EventListener != nil {
		err := t.components.EventListener.CleanUp()
		if err != nil {
			logger.Error(err, "")
		}
	}
	if t.components.SecretSync != nil {
		err := t.components.SecretSync.CleanUp()
		if err != nil {
			logger.Error(err, "")
		}
	}
}

// UpdateComponentCache will update the cache of the component to the cluster resource where the runtime is located.
func (t *Task) UpdateComponentCache() {
	ctx := context.TODO()
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.clusterName,
			Namespace: t.nautesNamespace,
		},
	}

	if err := t.tenantK8sClient.Get(ctx, client.ObjectKeyFromObject(cluster), cluster); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "get cluster failed")
		}
		return
	}

	_, err := controllerutil.CreateOrPatch(ctx, t.tenantK8sClient, cluster, func() error {
		if cluster.Status.ComponentsStatus == nil {
			cluster.Status.ComponentsStatus = map[string]pkgruntime.RawExtension{}
		}

		for componentType, status := range t.componentCache.status {
			rawCache, err := json.Marshal(status)
			if err != nil {
				return fmt.Errorf("convert component cache to string failed: %w", err)
			}
			cluster.Status.ComponentsStatus[string(componentType)] = pkgruntime.RawExtension{
				Raw: rawCache,
			}
		}

		return nil
	})
	if err != nil {
		logger.Error(err, "update component cache failed")
	}
}

// getClusterConnectInfo will obtain the access information of the cluster from the key management system and convert it to 'ClusterConnectInfo'.
func getClusterConnectInfo(ctx context.Context, secMgr component.SecretManagement) (*component.ClusterConnectInfo, error) {
	connectInfo, err := secMgr.GetAccessInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connect info failed: %w", err)
	}

	restConfig, err := kubeconvert.ConvertStringToRestConfig(connectInfo)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	return &component.ClusterConnectInfo{
		ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
		Kubernetes: &component.ClusterConnectInfoKubernetes{
			Config: restConfig,
		},
	}, nil
}

// initComponents creates components based on the cluster.
func initComponents(cii *component.ComponentInitInfo, cluster, _ *v1alpha1.Cluster, secMgr component.SecretManagement, statuses *ComponentsStatus) error {
	cii.Components.SecretManagement = secMgr

	components := cluster.Spec.ComponentsList

	if components.SecretSync != nil {
		newFn, ok := NewFunctionMapSecretSync[components.SecretSync.Name]
		if !ok {
			return fmt.Errorf("unknown secret sync type %s", components.SecretSync.Name)
		}

		secretSync, err := newFn(*components.SecretManagement, cii)
		if err != nil {
			return err
		}

		cii.Components.SecretSync = secretSync
	}

	if components.Deployment != nil {
		newFn, ok := NewFunctionMapDeployment[components.Deployment.Name]
		if !ok {
			return fmt.Errorf("unknown deployment type %s", components.Deployment.Name)
		}

		deployer, err := newFn(*components.Deployment, cii)
		if err != nil {
			return err
		}

		cii.Components.Deployment = deployer
	}

	if components.MultiTenant != nil {
		newFn, ok := NewFunctionMapMultiTenant[components.MultiTenant.Name]
		if !ok {
			return fmt.Errorf("unknown multi tenant type %s", components.MultiTenant.Name)
		}

		multiTenant, err := newFn(*components.MultiTenant, cii)
		if err != nil {
			return err
		}

		cii.Components.MultiTenant = multiTenant
	}

	if components.EventListener != nil {
		newFn, ok := NewFunctionMapEventListener[components.EventListener.Name]
		if !ok {
			return fmt.Errorf("unknown event listener type %s", components.EventListener.Name)
		}

		eventListener, err := newFn(*components.EventListener, cii)
		if err != nil {
			return err
		}

		cii.Components.EventListener = eventListener
	}

	if components.Pipeline != nil {
		factory, ok := ComponentFactoryMapPipeline[components.Pipeline.Name]
		if !ok {
			return fmt.Errorf("unknown pipeline type %s", components.Pipeline.Name)
		}

		rawStatus := GetRawStatus(component.ComponentTypePipeline, cluster.Status.ComponentsStatus)
		status, err := factory.NewStatus(rawStatus)
		if err != nil {
			return fmt.Errorf("get pipeline status failed: %w", err)
		}
		pipeline, err := factory.NewComponent(*components.Pipeline, cii, status)
		if err != nil {
			return err
		}
		statuses.status[component.ComponentTypePipeline] = status
		cii.Components.Pipeline = pipeline
	}

	return nil
}

func GetRawStatus(componentType component.ComponentType, statuses map[string]pkgruntime.RawExtension) []byte {
	if statuses == nil {
		return []byte{}
	}

	status, ok := statuses[string(componentType)]
	if !ok {
		return []byte{}
	}

	return status.Raw
}

func CreateProduct(ctx context.Context, productID, productName string, components component.ComponentList) error {
	req := component.PermissionRequest{
		RequestScope: component.RequestScopeProduct,
		Resource: component.ResourceMetaData{
			Product: productID,
			Name:    productID,
		},
		User:       productName,
		Permission: component.Permission{},
	}

	if err := components.MultiTenant.CreateProduct(ctx, productID); err != nil {
		return fmt.Errorf("create product in multi tenant failed: %w", err)
	}
	if err := components.Deployment.CreateProduct(ctx, productID); err != nil {
		return fmt.Errorf("create product in deployment failed: %w", err)
	}

	if err := components.Deployment.AddProductUser(ctx, req); err != nil {
		return fmt.Errorf("grant deployment product %s admin permission to group %s failed: %w",
			productID, productName, err)
	}

	return nil
}

func DeleteProduct(ctx context.Context, productID, productName string, components component.ComponentList) error {
	req := component.PermissionRequest{
		RequestScope: component.RequestScopeProduct,
		Resource: component.ResourceMetaData{
			Product: productID,
			Name:    productID,
		},
		User:       productName,
		Permission: component.Permission{},
	}

	if err := components.Deployment.DeleteProductUser(ctx, req); err != nil {
		return fmt.Errorf("revoke deployment product %s admin permission from group %s failed: %w",
			productID, productName, err)
	}

	if err := components.MultiTenant.DeleteProduct(ctx, productID); err != nil {
		return fmt.Errorf("create product in multi tenant failed: %w", err)
	}

	if err := components.Deployment.DeleteProduct(ctx, productID); err != nil {
		return fmt.Errorf("delete product in deployment failed: %w", err)
	}
	return nil
}

func DeleteProductAccount(ctx context.Context, account component.MachineAccount, components component.ComponentList) error {
	if account.Product == "" {
		return fmt.Errorf("account %s is not belongs to product", account.Name)
	}

	if err := components.SecretManagement.DeleteAccount(ctx, account); err != nil && !component.IsAccountNotFound(err) {
		return fmt.Errorf("delete account %s in secret management failed: %w", account.Name, err)
	}
	if err := components.MultiTenant.DeleteAccount(ctx, account.Product, account.Name); err != nil && !component.IsAccountNotFound(err) {
		return fmt.Errorf("delete account %s in multi tenant failed: %w", account.Name, err)
	}
	return nil
}

func AccountIsRemovable(usage cache.AccountUsage, accountName, runtimeName string) bool {
	runtimeUsage, ok := usage.Accounts[accountName]
	if !ok {
		return true
	}
	if len(runtimeUsage.Runtimes) == 0 {
		return true
	}
	if len(runtimeUsage.Runtimes) == 1 {
		if _, ok := runtimeUsage.Runtimes[runtimeName]; ok {
			return true
		}
	}
	return false
}

// syncAccountPermissionsInSecretManagement will update the permissions of the account in secret management based on the provided permission information.
func syncAccountPermissionsInSecretManagement(
	ctx context.Context,
	account component.MachineAccount,
	newPermissions, oldPermissions []component.SecretInfo,
	components component.ComponentList,
	runtimeName string,
	usage *cache.AccountUsage,
) error {
	for _, secretInfo := range newPermissions {
		if err := components.SecretManagement.GrantPermission(ctx, secretInfo, account); err != nil {
			return fmt.Errorf("grant permission to account %s failed: %w", account.Name, err)
		}
	}

	usage.UpdatePermissions(account.Name, runtimeName, newPermissions)
	accountPermissions := usage.Accounts[account.Name].ListPermissions()

	_, accountPermissionIndex := utils.GetHashMap(accountPermissions)
	oldPermissionMap, oldPermissionIndex := utils.GetHashMap(oldPermissions)
	revokePermissionIndexes := oldPermissionIndex.Difference(accountPermissionIndex)
	for _, index := range revokePermissionIndexes.UnsortedList() {
		if err := components.SecretManagement.RevokePermission(ctx, oldPermissionMap[index], account); err != nil {
			return fmt.Errorf("revoke permission to account %s failed: %w", account.Name, err)
		}
	}

	return nil
}

// revokeAccountPermissionsInSecretManagement will clear the permissions of the account in secret management based on the provided permission information.
func revokeAccountPermissionsInSecretManagement(
	ctx context.Context,
	account component.MachineAccount,
	permissions []component.SecretInfo,
	components component.ComponentList,
	runtimeName string,
	usage cache.AccountUsage,
) error {
	accountPermissions := usage.Accounts[account.Name].ListPermissions(cache.ExcludedRuntimes([]string{runtimeName}))

	_, apIndex := utils.GetHashMap(accountPermissions)
	oldPermissionMap, oldPermissionIndex := utils.GetHashMap(permissions)
	revokePermissionIndexes := oldPermissionIndex.Difference(apIndex)
	for _, index := range revokePermissionIndexes.UnsortedList() {
		if err := components.SecretManagement.RevokePermission(ctx, oldPermissionMap[index], account); err != nil {
			return fmt.Errorf("revoke permission to account %s failed: %w", account.Name, err)
		}
	}

	return nil
}
