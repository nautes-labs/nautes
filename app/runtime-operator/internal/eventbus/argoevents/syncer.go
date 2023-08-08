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

package argoevents

import (
	"context"
	"fmt"

	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Syncer struct {
	k8sClient client.Client
}

func NewSyncer(client client.Client) interfaces.EventBus {
	return Syncer{
		k8sClient: client,
	}
}

// SyncEvents used to make sure event bus is the same as project runtime define in dest cluster.
// It will create argo eventsource and sensor.
func (s Syncer) SyncEvents(ctx context.Context, task interfaces.RuntimeSyncTask) (*interfaces.EventBusDeploymentResult, error) {
	taskSyncer, err := newRuntimeSyncer(ctx, task, s.k8sClient)
	if err != nil {
		return nil, fmt.Errorf("create runtime syncer failed: %w", err)
	}

	err = taskSyncer.InitSecretRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("init secret repo for argo event failed: %w", err)
	}

	err = taskSyncer.SyncEventSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync event source failed: %w", err)
	}

	err = taskSyncer.SyncSensors(ctx)
	if err != nil {
		return nil, fmt.Errorf("sync sensor failed: %w", err)
	}
	return nil, nil
}

// RemoveEvents make sure that event bus defined in project pipeline runtime is not existed any more.
func (s Syncer) RemoveEvents(ctx context.Context, task interfaces.RuntimeSyncTask) error {
	taskSyncer, err := newRuntimeSyncer(ctx, task, s.k8sClient)
	if err != nil {
		return fmt.Errorf("create runtime syncer failed: %w", err)
	}

	if err := taskSyncer.DeleteSensors(ctx); err != nil {
		return fmt.Errorf("delete sensors failed: %w", err)
	}

	if err := taskSyncer.DeleteEventSources(ctx); err != nil {
		return fmt.Errorf("delete events source failed: %w", err)
	}

	return nil
}
