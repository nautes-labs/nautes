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

package deploymentruntime

import (
	"context"
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/constant"
	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	DeployApps = make(map[nautescfg.DeployAppType]interfaces.Deployment)
	EnvManagers = make(map[nautescrd.ClusterKind]interfaces.EnvManager)
	Pipelines = make(map[nautescfg.PipelineType]interfaces.Pipeline)
	EventBus = make(map[nautescfg.EventBusType]interfaces.EventBus)
}

var (
	DeployApps  map[nautescfg.DeployAppType]interfaces.Deployment
	EnvManagers map[nautescrd.ClusterKind]interfaces.EnvManager
	Pipelines   map[nautescfg.PipelineType]interfaces.Pipeline
	EventBus    map[nautescfg.EventBusType]interfaces.EventBus
)

var (
	logger = log.Log.WithName("runtime-syncer")
)

type Syncer struct {
	Client client.Client
}

// Sync will build up runtime in destination
func (s *Syncer) Sync(ctx context.Context, runtime interfaces.Runtime) (*interfaces.RuntimeDeploymentResult, error) {
	cfg, ok := runtimecontext.FromNautesConfigContext(ctx)
	if !ok {
		return nil, fmt.Errorf("get nautes config from context failed")
	}

	productDB, err := database.NewRuntimeDataSource(ctx, s.Client, runtime.GetProduct(), cfg.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("collect product infos failed: %w", err)
	}

	cluster, err := productDB.GetClusterByRuntime(runtime.(nautescrd.Runtime))
	if err != nil {
		return nil, fmt.Errorf("get cluster info failed: %w", err)
	}

	var hostCluster *nautescrd.Cluster
	if cluster.Spec.ClusterType == nautescrd.CLUSTER_TYPE_VIRTUAL {
		hostCluster = &nautescrd.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Spec.HostCluster,
				Namespace: cluster.Namespace,
			},
		}
	}

	result := &interfaces.RuntimeDeploymentResult{
		Cluster: cluster.Name,
		App:     interfaces.SelectedApp{},
	}

	envManager, ok := EnvManagers[cluster.Spec.ClusterKind]
	if !ok {
		return nil, fmt.Errorf("cluster provider %s is not support", cluster.Spec.ClusterKind)
	}

	accessInfo, err := envManager.GetAccessInfo(ctx, *cluster)
	if err != nil {
		return nil, fmt.Errorf("get access info failed: %w", err)
	}

	deployTask, err := s.createDeployTask(cfg, runtime, func(t *interfaces.RuntimeSyncTask) {
		t.AccessInfo = *accessInfo
		t.Cluster = *cluster
		t.HostCluster = hostCluster
	})
	if err != nil {
		return nil, fmt.Errorf("create deploy task failed: %w", err)
	}

	_, err = envManager.Sync(ctx, *deployTask)
	if err != nil {
		return nil, fmt.Errorf("init env failed: %w", err)
	}

	initInfo, err := newComponentInitInfo(runtime.(nautescrd.Runtime), productDB, accessInfo.Kubernetes)
	if err != nil {
		return nil, fmt.Errorf("get deploy task failed: %w", err)
	}

	componentDeployment := cluster.Spec.ComponentsList.Deployment
	if componentDeployment == nil {
		return nil, fmt.Errorf("cluster %s deployment component is nil", cluster.Name)
	}

	runtimeSyncer, err := NewRuntimeSyncer(ctx, *initInfo, productDB, cfg)
	if err != nil {
		return nil, fmt.Errorf("create runtime syncer failed: %w", err)
	}

	deployer, err := newDeployer(*componentDeployment, *initInfo, productDB)
	if err != nil {
		return nil, fmt.Errorf("get deployer failed: %w", err)
	}

	app, err := newDeployAppV2(runtime.(nautescrd.Runtime), productDB)
	if err != nil {
		return nil, fmt.Errorf("get deploy app failed: %w", err)
	}

	switch deployTask.RuntimeType {
	case interfaces.RUNTIME_TYPE_DEPLOYMENT:
		deployApp, err := getDeployApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err != nil {
			return nil, err
		}

		AppDeployInfo, err := deployApp.Deploy(ctx, *deployTask)
		if err != nil {
			return nil, fmt.Errorf("deploy app failed: %w", err)
		}
		result.DeploymentDeploymentResult = AppDeployInfo

	case interfaces.RUNTIME_TYPE_PIPELINE:
		eventBusApp, err := getEventBusApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err != nil {
			return nil, err
		}

		_, err = eventBusApp.SyncEvents(ctx, *deployTask)
		if err != nil {
			return nil, fmt.Errorf("sync event failed: %w", err)
		}

		pipelineApp, err := getPipelineApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err := pipelineApp.DeployPipelineRuntime(ctx, *deployTask); err != nil {
			return nil, fmt.Errorf("sync pipeline runtime failed: %w", err)
		}

		if err := runtimeSyncer.SyncCodeRepos(ctx); err != nil {
			return nil, fmt.Errorf("sync code repos failed: %w", err)
		}

		if app != nil {
			err := deployer.DeployApp(ctx, *app)
			if err != nil {
				return nil, fmt.Errorf("deploy app failed: %w", err)
			}
		}
	}
	return result, nil
}

// Delete will clean up runtime in destination
func (s *Syncer) Delete(ctx context.Context, runtime interfaces.Runtime) error {
	cfg, ok := runtimecontext.FromNautesConfigContext(ctx)
	if !ok {
		return fmt.Errorf("get nautes config from context failed")
	}

	productDB, err := database.NewRuntimeDataSource(ctx, s.Client, runtime.GetProduct(), cfg.Nautes.Namespace)
	if err != nil {
		return fmt.Errorf("collect product infos failed: %w", err)
	}

	cluster, err := productDB.GetClusterByRuntime(runtime.(nautescrd.Runtime))
	if err != nil {
		return fmt.Errorf("get cluster info failed: %w", err)
	}

	envManager, ok := EnvManagers[cluster.Spec.ClusterKind]
	if !ok {
		return fmt.Errorf("cluster provider %s is not support", cluster.Spec.ClusterKind)
	}

	accessInfo, err := envManager.GetAccessInfo(ctx, *cluster)
	if err != nil {
		return fmt.Errorf("get access info failed: %w", err)
	}

	deployTask, err := s.createDeployTask(cfg, runtime, func(t *interfaces.RuntimeSyncTask) {
		t.AccessInfo = *accessInfo
		t.Cluster = *cluster
	})
	if err != nil {
		return fmt.Errorf("create deploy task failed: %w", err)
	}

	initInfo, err := newComponentInitInfo(runtime.(nautescrd.Runtime), productDB, accessInfo.Kubernetes)
	if err != nil {
		return fmt.Errorf("get deploy task failed: %w", err)
	}

	componentDeployment := cluster.Spec.ComponentsList.Deployment
	if componentDeployment == nil {
		return fmt.Errorf("cluster %s deployment component is nil", cluster.Name)
	}

	runtimeSyncer, err := NewRuntimeSyncer(ctx, *initInfo, productDB, cfg)
	if err != nil {
		return fmt.Errorf("create runtime syncer failed: %w", err)
	}

	deployer, err := newDeployer(*componentDeployment, *initInfo, productDB)
	if err != nil {
		return fmt.Errorf("get deployer failed: %w", err)
	}

	app, err := newDeployAppV2(runtime.(nautescrd.Runtime), productDB)
	if err != nil {
		return fmt.Errorf("get deploy app failed: %w", err)
	}

	switch deployTask.RuntimeType {
	case interfaces.RUNTIME_TYPE_DEPLOYMENT:
		deployApp, err := getDeployApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err != nil {
			return err
		}

		err = deployApp.UnDeploy(ctx, *deployTask)
		if err != nil {
			return fmt.Errorf("remove app failed: %w", err)
		}
	case interfaces.RUNTIME_TYPE_PIPELINE:
		EventBusApp, err := getEventBusApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err != nil {
			return err
		}
		err = EventBusApp.RemoveEvents(ctx, *deployTask)
		if err != nil {
			return fmt.Errorf("sync event failed: %w", err)
		}
		pipelineApp, err := getPipelineApp(ctx, string(cluster.Spec.ClusterKind), cfg)
		if err := pipelineApp.UnDeployPipelineRuntime(ctx, *deployTask); err != nil {
			return fmt.Errorf("sync pipeline runtime failed: %w", err)
		}

		if app != nil {
			if err := deployer.UnDeployApp(ctx, *app); err != nil {
				return fmt.Errorf("remove deploy app failed: %w", err)
			}
		}

		if err := runtimeSyncer.SyncCodeRepos(ctx); err != nil {
			return fmt.Errorf("sync code repos failed: %w", err)
		}
	}

	err = envManager.Remove(ctx, *deployTask)
	if err != nil {
		return fmt.Errorf("remove env failed: %w", err)
	}

	return nil
}

func (s *Syncer) getCluster(ctx context.Context, productName, name string, nautesNamespace string) (*nautescrd.Cluster, error) {
	env := &nautescrd.Environment{}
	key := types.NamespacedName{
		Namespace: productName,
		Name:      name,
	}

	if err := s.Client.Get(ctx, key, env); err != nil {
		return nil, err
	}

	cluster := &nautescrd.Cluster{}
	key = types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      env.Spec.Cluster,
	}
	if err := s.Client.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

type setDeployTask func(*interfaces.RuntimeSyncTask)

func (s *Syncer) createDeployTask(cfg nautescfg.Config, runtime interfaces.Runtime, opts ...setDeployTask) (*interfaces.RuntimeSyncTask, error) {
	product := &nautescrd.Product{}
	key := types.NamespacedName{
		Namespace: cfg.Nautes.Namespace,
		Name:      runtime.GetProduct(),
	}
	if err := s.Client.Get(context.TODO(), key, product); err != nil {
		return nil, err
	}

	var runtimeType interfaces.RuntimeType
	switch runtime.(type) {
	case *nautescrd.DeploymentRuntime:
		runtimeType = interfaces.RUNTIME_TYPE_DEPLOYMENT
	case *nautescrd.ProjectPipelineRuntime:
		runtimeType = interfaces.RUNTIME_TYPE_PIPELINE
	default:
		return nil, fmt.Errorf("unknow runtime type")
	}
	logger.V(1).Info(fmt.Sprintf("runtime %s is a %s runtime", runtime.GetName(), runtimeType))

	task := &interfaces.RuntimeSyncTask{
		Product:            *product,
		NautesCfg:          cfg,
		Runtime:            runtime,
		RuntimeType:        runtimeType,
		ServiceAccountName: constant.ServiceAccountDefault,
	}

	for _, fn := range opts {
		fn(task)
	}

	return task, nil
}

func getDeployApp(ctx context.Context, clusterKind string, cfg nautescfg.Config) (interfaces.Deployment, error) {
	deployAppType, ok := cfg.Deploy.DefaultApp[clusterKind]
	if !ok {
		return nil, fmt.Errorf("unknow deployment type %s", clusterKind)
	}

	deployApp, ok := DeployApps[deployAppType]
	if !ok {
		return nil, fmt.Errorf("deployment type %s is not support", deployAppType)
	}

	return deployApp, nil
}

func getEventBusApp(ctx context.Context, clusterKind string, cfg nautescfg.Config) (interfaces.EventBus, error) {
	eventBusAppType, ok := cfg.EventBus.DefaultApp[clusterKind]
	if !ok {
		return nil, fmt.Errorf("unknow deployment type %s", clusterKind)
	}

	eventBusApp, ok := EventBus[eventBusAppType]
	if !ok {
		return nil, fmt.Errorf("deployment type %s is not support", eventBusAppType)
	}

	return eventBusApp, nil
}

func getPipelineApp(ctx context.Context, clusterKind string, cfg nautescfg.Config) (interfaces.Pipeline, error) {
	pipelineAppType, ok := cfg.Pipeline.DefaultApp[clusterKind]
	if !ok {
		return nil, fmt.Errorf("unknow deployment type %s", clusterKind)
	}

	pipelineApp, ok := Pipelines[pipelineAppType]
	if !ok {
		return nil, fmt.Errorf("deployment type %s is not support", pipelineAppType)
	}

	return pipelineApp, nil
}
