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

package middlewareruntime

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
)

type MiddlewareRuntimePerformer struct {
	runtime      *v1alpha1.MiddlewareRuntime
	environment  *v1alpha1.Environment
	currentState *v1alpha1.MiddlewareRuntimeStatus
	lastState    *v1alpha1.MiddlewareRuntimeStatus
}

// newPerformer creates a new task performer for the MiddlewareRuntimePerformer.
// It takes an initInfo parameter of type task.PerformerInitInfos.
// It returns a task.TaskPerformer and an error.
func newPerformer(initInfo task.PerformerInitInfos) (task.TaskPerformer, error) {
	performer := &MiddlewareRuntimePerformer{}
	return performer, nil
}

// Deploy loops through the MiddlewareRuntimeSpecs in the runtime and creates corresponding middleware for each Middleware in the MiddlewareRuntimeSpec.
func (m *MiddlewareRuntimePerformer) Deploy(ctx context.Context) (interface{}, error) {
	// 1. Get all middleware under the runtime in the performer.
	// 2. Calculate the list of all spaces.
	// 3. If the environment points to Cluster resources, append the following steps:
	//    - Create projects and runtime environments based on the runtime information.
	//    - Create network entry points for the services based on the entrypoint information.
	//
	// 4. Create MiddlewareDeployer.
	// 5. Deploy middleware using MiddlewareDeployer.
	//    - If the deployment is successful, update the middleware's status information.
	//
	// 6. Return the new status information.
	return m.currentState, nil
}

func (m *MiddlewareRuntimePerformer) Delete(ctx context.Context) (interface{}, error) {
	// Implement the Delete method
	return nil, nil
}
