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

package secretclient

import (
	"context"

	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type SecretClient interface {
	// Give history cluster and current cluster, this function should keep remote is the same as define.
	// It should return the record of sync result, if sync failed, should return nil status.
	Sync(ctx context.Context, cluster, lastCluster *clustercrd.Cluster) (*SyncResult, error)
	// Clean up remote by giving cluster, this cluster is the last success record, not the current info.
	Delete(ctx context.Context, cluster *clustercrd.Cluster) error
	GetKubeConfig(ctx context.Context, cluster *clustercrd.Cluster) (string, error)
	Logout()
}

type SyncResult struct {
	SecretID string
}
