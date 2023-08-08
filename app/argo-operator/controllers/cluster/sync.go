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
	"fmt"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kops/pkg/kubeconfig"
)

func (r *ClusterReconciler) saveChangesCluster(ctx context.Context, cluster *resourcev1alpha1.Cluster, kubeconfig *kubeconfig.KubectlConfig, preClusterServer string) error {
	clusterInfo, err := r.getClusterInfo(cluster.Spec.ApiServer)
	if err != nil {
		err = r.createCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return err
		}
	} else if err == nil && !clusterInfo.valid {
		err = r.updateCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return err
		}
	}

	err = r.deleteCluster(ctx, preClusterServer)
	if err != nil {
		return err
	}

	return nil
}

func (r *ClusterReconciler) saveClusterFromSecretChange(ctx context.Context, cluster *resourcev1alpha1.Cluster, kubeconfig *kubeconfig.KubectlConfig) error {
	_, err := r.getClusterInfo(cluster.Spec.ApiServer)
	if err != nil {
		err := r.createCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return err
		}

	} else {
		err = r.updateCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ClusterReconciler) regularSaveCluster(ctx context.Context, cluster *resourcev1alpha1.Cluster, kubeconfig *kubeconfig.KubectlConfig) (bool, error) {
	clusterInfo, err := r.getClusterInfo(cluster.Spec.ApiServer)
	if err != nil {
		err := r.createCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err == nil && !clusterInfo.valid {
		err = r.updateCluster(ctx, cluster, kubeconfig)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *ClusterReconciler) syncCluster2Argocd(ctx context.Context, cluster *resourcev1alpha1.Cluster, kubeconfig *kubeconfig.KubectlConfig, secretID string) (bool, error) {
	var tune = true
	var sync = false

	preClusterServer, clusterChange := r.isClusterChange(cluster)
	if clusterChange {
		err := r.saveChangesCluster(ctx, cluster, kubeconfig, preClusterServer)
		if err != nil {
			return !tune, err
		}
		sync = true
	} else {
		secretChange := r.isSecretChange(cluster, secretID)
		if secretChange {
			if err := r.saveClusterFromSecretChange(ctx, cluster, kubeconfig); err != nil {
				return !tune, err
			}
			sync = true
		} else {
			save, err := r.regularSaveCluster(ctx, cluster, kubeconfig)
			if err != nil {
				return !tune, err
			}
			sync = save
		}
	}

	if sync {
		if err := r.updateSync2ArgoStatus(ctx, cluster.Spec, secretID); err != nil {
			return tune, err
		}

		message := fmt.Sprintf("successfully saved cluster %s to argocd", cluster.Name)
		condition = metav1.Condition{Type: ClusterConditionType, Message: message, Reason: RegularUpdate, Status: metav1.ConditionTrue}
		if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
			return tune, err
		}
	}

	return tune, nil
}
