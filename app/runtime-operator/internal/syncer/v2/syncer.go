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
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	kubeconvert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	NewFunctionMapDeployment       = map[string]NewDeployment{}
	NewFunctionMapMultiTenant      = map[string]NewMultiTenant{}
	NewFunctionMapSecretManagement = map[string]NewSecretManagement{}
	NewFunctionMapSecretSync       = map[string]NewSecretSync{}
	NewFunctionMapGateway          = map[string]NewGateway{}
	NewFunctionMapEventListener    = map[string]NewEventListener{}
)

var (
	newFunctionMapTaskRunner = map[v1alpha1.RuntimeType]NewTaskRuner{
		v1alpha1.RuntimeTypeDeploymentRuntime: newDeploymentRuntimeDeployer,
		v1alpha1.RuntimeTypePipelineRuntime:   newPipelineRuntimeDeployer,
	}
	NewDatabase = database.NewRuntimeDataSource
)

var (
	logger = logf.Log.WithName("taskRunner")
)

type NewTaskRuner func(initInfo runnerInitInfos) (taskRunner, error)

type runnerInitInfos struct {
	*ComponentInitInfo
	runtime         v1alpha1.Runtime
	cache           *pkgruntime.RawExtension
	tenantK8sClient client.Client
}

type taskRunner interface {
	Deploy(ctx context.Context) (*pkgruntime.RawExtension, error)
	Delete(ctx context.Context) (*pkgruntime.RawExtension, error)
}

type Syncer struct {
	KubernetesClient client.Client
}

type Task struct {
	components *ComponentList
	runner     taskRunner
}

// NewTasks loads nautes resources from tenant cluster and initializes components.
func (s *Syncer) NewTasks(ctx context.Context, runtime v1alpha1.Runtime, cache *pkgruntime.RawExtension) (*Task, error) {
	cfg, err := configs.NewNautesConfigFromFile()
	if err != nil {
		return nil, fmt.Errorf("load nautes config failed: %w", err)
	}

	productName := runtime.GetProduct()
	db, err := NewDatabase(ctx, s.KubernetesClient, productName, cfg.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("init nautes database failed: %w", err)
	}

	cluster, err := db.GetClusterByRuntime(runtime)
	if err != nil {
		return nil, fmt.Errorf("get cluster by runtime failed: %w", err)
	}

	var hostCluster *v1alpha1.Cluster
	if cluster.Spec.ClusterType == v1alpha1.CLUSTER_TYPE_VIRTUAL {
		cluster, err := db.GetCluster(cluster.Spec.HostCluster)
		if err != nil {
			return nil, fmt.Errorf("get host cluster failed: %w", err)
		}
		hostCluster = cluster
	}

	initInfo := &ComponentInitInfo{
		ClusterConnectInfo: ClusterConnectInfo{},
		ClusterName:        cluster.GetName(),
		RuntimeType:        v1alpha1.RuntimeTypeDeploymentRuntime,
		NautesDB:           db,
		NautesConfig:       *cfg,
		Components: &ComponentList{
			Deployment:       nil,
			MultiTenant:      nil,
			SecretManagement: nil,
			SecretSync:       nil,
			EventListener:    nil,
		},
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

	if err := initInfo.initComponents(cluster, hostCluster, secMgr); err != nil {
		return nil, fmt.Errorf("init componentList failed: %w", err)
	}

	runnerInitInfos := runnerInitInfos{
		ComponentInitInfo: initInfo,
		runtime:           runtime,
		cache:             cache,
		tenantK8sClient:   s.KubernetesClient,
	}

	runner, err := newFunctionMapTaskRunner[runtime.GetRuntimeType()](runnerInitInfos)
	if err != nil {
		return nil, err
	}

	return &Task{
		components: initInfo.Components,
		runner:     runner,
	}, nil
}

const (
	ConfigMapNameClustersUsageCache = "clusters-usage-cache"
)

func (t *Task) Run(ctx context.Context) (*pkgruntime.RawExtension, error) {
	defer t.CleanUp()
	return t.runner.Deploy(ctx)
}

func (t *Task) Delete(ctx context.Context) (*pkgruntime.RawExtension, error) {
	defer t.CleanUp()
	return t.runner.Delete(ctx)
}

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

// getClusterConnectInfo retrieves connect info from cluster resource.
func getClusterConnectInfo(ctx context.Context, secMgr SecretManagement) (*ClusterConnectInfo, error) {
	connectInfo, err := secMgr.GetAccessInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get connect info failed: %w", err)
	}

	restConfig, err := kubeconvert.ConvertStringToRestConfig(connectInfo)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	return &ClusterConnectInfo{
		Type: v1alpha1.CLUSTER_KIND_KUBERNETES,
		Kubernetes: &ClusterConnectInfoKubernetes{
			Config: restConfig,
		},
	}, nil
}

// initComponents creates components based on the cluster.
func (cii *ComponentInitInfo) initComponents(cluster, _ *v1alpha1.Cluster, secMgr SecretManagement) error {
	components := cluster.Spec.ComponentsList

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

	cii.Components.SecretManagement = secMgr

	return nil
}
