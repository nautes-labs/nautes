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
	"encoding/base64"
	"encoding/json"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kops/pkg/kubeconfig"
)

// isClusterChange if cluster resource url has been changed, return previous cluster url and true
func (r *ClusterReconciler) isClusterChange(cluster *resourcev1alpha1.Cluster) (string, bool) {
	if cluster.Status.Sync2ArgoStatus != nil && cluster.Status.Sync2ArgoStatus.LastSuccessSpec != "" {
		lastSpecStr := cluster.Status.Sync2ArgoStatus.LastSuccessSpec
		lastSpec := &resourcev1alpha1.ClusterSpec{}
		if lastSpecStr != "" {
			if err := json.Unmarshal([]byte(lastSpecStr), lastSpec); err != nil {
				return "", false
			}
		}
		if lastSpec != nil && lastSpec.ApiServer != cluster.Spec.ApiServer {
			return lastSpec.ApiServer, true
		}
	}

	return "", false
}

func (r *ClusterReconciler) getClusterInfo(clusterAddress string) (*clusterInfo, error) {
	if err := r.Argocd.Auth().Login(); err != nil {
		return nil, err
	}

	response, err := r.Argocd.Cluster().GetClusterInfo(clusterAddress)
	if err != nil {
		return nil, err
	}

	valid := response.ConnectionState.Status == "Successful" || response.ConnectionState.Status == "Unknown"
	return &clusterInfo{
		name:   response.Name,
		server: response.Server,
		valid:  valid,
	}, nil
}

func (r *ClusterReconciler) getClusterResource(ctx context.Context, namespacedName types.NamespacedName) (*resourcev1alpha1.Cluster, error) {
	cluster := &resourcev1alpha1.Cluster{}
	err := r.Get(ctx, namespacedName, cluster)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

type clusterInfo struct {
	name   string
	server string
	valid  bool
}

func (r *ClusterReconciler) createCluster(ctx context.Context, cluster *resourcev1alpha1.Cluster, cfg *kubeconfig.KubectlConfig) error {
	cientCertificateData := base64.StdEncoding.EncodeToString(cfg.Clusters[0].Cluster.CertificateAuthorityData)
	clientCertificateData := base64.StdEncoding.EncodeToString(cfg.Users[0].User.ClientCertificateData)
	clientKeyData := base64.StdEncoding.EncodeToString(cfg.Users[0].User.ClientKeyData)

	toCreate := &argocd.ClusterInstance{
		Name:            cluster.ObjectMeta.Name,
		ApiServer:       cluster.Spec.ApiServer,
		HostCluster:     cluster.Spec.HostCluster,
		CaData:          cientCertificateData,
		CertificateData: clientCertificateData,
		KeyData:         clientKeyData,
		Usage:           cluster.Spec.Usage,
	}

	err := r.Argocd.Auth().Login()
	if err != nil {
		return err
	}
	err = r.Argocd.Cluster().CreateCluster(toCreate)
	if err != nil {
		condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
		if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *ClusterReconciler) updateCluster(ctx context.Context, cluster *resourcev1alpha1.Cluster, cfg *kubeconfig.KubectlConfig) error {
	cientCertificateData := base64.StdEncoding.EncodeToString(cfg.Clusters[0].Cluster.CertificateAuthorityData)
	clientCertificateData := base64.StdEncoding.EncodeToString(cfg.Users[0].User.ClientCertificateData)
	clientKeyData := base64.StdEncoding.EncodeToString(cfg.Users[0].User.ClientKeyData)

	toUpdate := &argocd.ClusterInstance{
		Name:            cluster.ObjectMeta.Name,
		ApiServer:       cluster.Spec.ApiServer,
		HostCluster:     cluster.Spec.HostCluster,
		CaData:          cientCertificateData,
		CertificateData: clientCertificateData,
		KeyData:         clientKeyData,
		Usage:           cluster.Spec.Usage,
	}

	if err := r.Argocd.Auth().Login(); err != nil {
		return err
	}
	if err := r.Argocd.Cluster().UpdateCluster(toUpdate); err != nil {
		condition = metav1.Condition{Type: ClusterConditionType, Message: err.Error(), Reason: RegularUpdate, Status: metav1.ConditionFalse}
		if err := r.setConditionAndUpdateStatus(ctx, condition); err != nil {
			return err
		}

		return err
	}

	return nil
}

func (r *ClusterReconciler) deleteCluster(_ context.Context, apiServer string) error {
	err := r.Argocd.Auth().Login()
	if err != nil {
		return err
	}

	clusterInfo, err := r.Argocd.Cluster().GetClusterInfo(apiServer)
	if err != nil {
		if _, ok := err.(*argocd.IsNoAuth); ok {
			return nil
		}
		return err
	}

	_, err = r.Argocd.Cluster().DeleteCluster(clusterInfo.Server)
	if err != nil {
		return err
	}

	return nil
}
