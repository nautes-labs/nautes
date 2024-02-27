// Copyright 2024 Nautes Authors
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

package performer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTaskPerformer will generate a taskPerformer instance.
type NewTaskPerformer func(initInfo PerformerInitInfos) (TaskPerformer, error)

// PerformerInitInfos stores the information needed to initialize taskPerformer.
type PerformerInitInfos struct {
	*component.ComponentInitInfo
	// Runtime is the Runtime resource to be deployed.
	Runtime v1alpha1.Runtime
	// Cache is the information last deployed on the runtime.
	Cache *pkgruntime.RawExtension
	// TenantK8sClient is the k8s client of tenant cluster.
	TenantK8sClient client.Client
}

// TaskPerformer can deploy or clean up specific types of runtime.
type TaskPerformer interface {
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
