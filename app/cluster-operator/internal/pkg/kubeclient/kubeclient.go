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

package kubernetesclient

import (
	"context"

	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	kubeconvert "github.com/nautes-labs/nautes/pkg/kubeconvert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	ClusterGroupResrource = schema.GroupVersionResource{
		Group:    clustercrd.GroupVersion.Group,
		Version:  clustercrd.GroupVersion.Version,
		Resource: "clusters",
	}
)

type K8SClient struct {
	kubernetes.Clientset
	Dynamic dynamic.Interface
}

func NewK8SClient(kubeconfig string) (*K8SClient, error) {
	var client *kubernetes.Clientset
	var dclient dynamic.Interface
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = ctrl.GetConfig()
	} else {
		config, err = kubeconvert.ConvertStringToRestConfig(kubeconfig)
	}
	if err != nil {
		return nil, err
	}

	client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dclient, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8SClient{
		Clientset: *client,
		Dynamic:   dclient,
	}, nil
}

func (k *K8SClient) GetCluster(ctx context.Context, name, namespace string) (*clustercrd.Cluster, error) {
	resp, err := k.Dynamic.Resource(ClusterGroupResrource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	clus := &clustercrd.Cluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(resp.UnstructuredContent(), clus)
	if err != nil {
		return nil, err
	}

	return clus, nil
}

func (k *K8SClient) UpdateCluster(ctx context.Context, cluster *clustercrd.Cluster) error {
	res := &unstructured.Unstructured{}
	res_stringamap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return err
	}
	res.SetUnstructuredContent(res_stringamap)

	_, err = k.Dynamic.Resource(ClusterGroupResrource).Namespace(cluster.Namespace).Update(ctx, res, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *K8SClient) UpdateStatusCluster(ctx context.Context, cluster *clustercrd.Cluster) error {
	res := &unstructured.Unstructured{}
	res_stringamap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return err
	}
	res.SetUnstructuredContent(res_stringamap)

	_, err = k.Dynamic.Resource(ClusterGroupResrource).Namespace(cluster.Namespace).UpdateStatus(ctx, res, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *K8SClient) GetServiceAccount(ctx context.Context, name, namespace string) (*v1.ServiceAccount, error) {
	return k.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *K8SClient) GetSecret(ctx context.Context, name, namespace string) (*v1.Secret, error) {
	return k.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *K8SClient) ListStatefulSets(ctx context.Context, namespace string, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	return k.AppsV1().StatefulSets(namespace).List(ctx, opts)
}
