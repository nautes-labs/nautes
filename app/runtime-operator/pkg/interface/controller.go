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

package interfaces

import (
	"context"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

// RuntimeSyncer is use to deploy or clean up runtime in dest environment. It can handle any type of runtime.
type RuntimeSyncer interface {
	Sync(ctx context.Context, runtime Runtime) (*RuntimeDeploymentResult, error)
	Delete(ctx context.Context, runtime Runtime) error
}

// RuntimeSyncTask stores all the information needed during the deployment processe.
type RuntimeSyncTask struct {
	// AccessInfo use to connect dest environment, like kubeconfig of kubernetes.
	AccessInfo AccessInfo
	// Product store the runtime's product resource.
	Product nautescrd.Product
	// Cluster store runtime's cluster resource.
	Cluster     nautescrd.Cluster
	HostCluster *nautescrd.Cluster
	NautesCfg   nautescfg.Config
	// Runtime is the runtime resource of this sync, it chould be pipeline runtime or deployment runtime.
	Runtime     Runtime
	RuntimeType RuntimeType
	// ServiceAccountName is the authorized account name in k8s, it can get secrets from secret store, create resource in runtime namespace, etc.
	ServiceAccountName string
}

type RuntimeDeploymentResult struct {
	Cluster                     string
	App                         SelectedApp
	DeploymentDeploymentResult  *DeploymentDeploymentResult
	EnvironmentDeploymentResult *EnvironmentDeploymentResult
	EventBusDeploymentResult    *EventBusDeploymentResult
}

// SelectedApp records the specific app name used during deployment.
type SelectedApp struct {
	Pipeline    string
	Deploy      string
	EventBus    string
	Environment string
}

func (t *RuntimeSyncTask) GetLabel() map[string]string {
	return map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: t.Product.Name}
}
