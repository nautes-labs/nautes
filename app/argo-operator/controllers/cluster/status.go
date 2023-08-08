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

package cluster

import (
	"context"
	"encoding/json"
	"time"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ClusterReconciler) updateSync2ArgoStatus(ctx context.Context, spec resourcev1alpha1.ClusterSpec, secretID string) error {
	cluster, err := r.getClusterResource(ctx, namespacedName)
	if err != nil {
		return err
	}

	if cluster.Status.Sync2ArgoStatus == nil {
		cluster.Status.Sync2ArgoStatus = &resourcev1alpha1.SyncCluster2ArgoStatus{}
	}

	if cluster != nil {
		jsonSpec, err := json.Marshal(spec)
		if err != nil {
			condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
			if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
				return err
			}

			return err
		}
		cluster.Status.Sync2ArgoStatus.LastSuccessSpec = string(jsonSpec)
		cluster.Status.Sync2ArgoStatus.SecretID = secretID
		cluster.Status.Sync2ArgoStatus.LastSuccessTime = metav1.NewTime(time.Now())
	}

	if err = r.Status().Update(ctx, cluster); err != nil {
		return err
	}

	return nil
}

func (r *ClusterReconciler) setConditionAndUpdateStatus(ctx context.Context, condition metav1.Condition) error {
	cluster, err := r.getClusterResource(ctx, namespacedName)
	if err != nil {
		return err
	}

	cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{condition.Type: true})

	conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
	if len(conditions) > 0 {
		if cluster.Status.Sync2ArgoStatus != nil && conditions[0].Status == metav1.ConditionTrue {
			cluster.Status.Sync2ArgoStatus.LastSuccessTime = conditions[0].LastTransitionTime
		}
	}

	err = r.Status().Update(ctx, cluster)
	if err != nil {
		return err
	}

	return nil
}

func (r *ClusterReconciler) appendFinalizerAndUpdateStaus(ctx context.Context, FinalizerName string, namespacedName types.NamespacedName) error {
	cluster, err := r.getClusterResource(ctx, namespacedName)
	if err != nil {
		return err
	}

	cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, FinalizerName)
	if err := r.Update(context.Background(), cluster); err != nil {
		return err
	}

	return nil
}
