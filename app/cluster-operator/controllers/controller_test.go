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

package controllers

import (
	"context"
	"errors"
	"time"

	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterConfig "github.com/nautes-labs/nautes/pkg/config/cluster"
)

var _ = Describe("Reconcile", func() {
	var cluster *clustercrd.Cluster

	BeforeEach(func() {
		cluster = &clustercrd.Cluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "nautes",
				Annotations: map[string]string{
					"ver": "1",
				},
			},
			Spec: clustercrd.ClusterSpec{
				ApiServer:                         "https://127.0.0.1",
				ClusterType:                       clustercrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind:                       clustercrd.CLUSTER_KIND_KUBERNETES,
				Usage:                             clustercrd.CLUSTER_USAGE_HOST,
				HostCluster:                       "",
				ReservedNamespacesAllowedProducts: make(map[string][]string),
				ProductAllowedClusterResources:    make(map[string][]clustercrd.ClusterResourceInfo),
				ComponentsList: clustercrd.ComponentsList{
					Gateway: &clustercrd.Component{
						Name:      "traefik",
						Namespace: "traefik",
						Additions: map[string]string{
							"httpsNodePort": "30020",
							"httpNodePort":  "30221",
						},
					},
					OauthProxy: &clustercrd.Component{
						Name:      "oauth2-proxy",
						Namespace: "oauth2-proxy",
					},
					CertManagement:      &clustercrd.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &clustercrd.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &clustercrd.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &clustercrd.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &clustercrd.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &clustercrd.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &clustercrd.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &clustercrd.Component{Name: "external-secrets", Namespace: "external-secrets"},
				},
			},
		}
		wantErr = nil
		wantResult = &secretclient.SyncResult{
			SecretID: "1",
		}
	})

	BeforeEach(func() {
		err := clusterConfig.SetClusterValidateConfig()
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		wantErr = nil
		err := k8sClient.Delete(context.Background(), cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = WaitForDelete(cluster)
		Expect(err).Should(BeNil())

		err = clusterConfig.DeleteValidateConfig()
		Expect(err).Should(BeNil())
	})

	It("sync secret", func() {
		startTime := time.Now()
		err := k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeSecretStore)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("True"))

		nc := &clustercrd.Cluster{}
		err = k8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, nc)
		Expect(err).Should(BeNil())
		Expect(nc.Status.MgtAuthStatus.SecretID).Should(Equal("1"))
	})

	It("sync failed", func() {
		startTime := time.Now()
		wantErr = errors.New("new error")
		err := k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeSecretStore)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))
	})

	It("object is not legal", func() {
		startTime := time.Now()
		cluster.Spec.HostCluster = "newCluster"
		err := k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeSecretStore)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))

	})

	It("delete failed", func() {
		startTime := time.Now()
		time.Sleep(time.Second)
		err := k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeSecretStore)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("True"))

		startTime = time.Now()
		time.Sleep(time.Second)
		wantErr = errors.New("new error")
		k8sClient.Delete(context.Background(), cluster)
		cdt, err = WaitForCondition(cluster, startTime, clusterConditionTypeSecretStore)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))
	})

})

var _ = Describe("Reconcile", func() {
	var cluster *clustercrd.Cluster
	var hostCluster *clustercrd.Cluster
	var svc *corev1.Service
	BeforeEach(func() {
		svc = &corev1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name:      "steam",
				Namespace: "default",
				Labels:    map[string]string{"resource.nautes.io/Usage": "Entrypoint"},
				Annotations: map[string]string{
					"cluster-operator.nautes.io/http-entrypoint":  "http",
					"cluster-operator.nautes.io/https-entrypoint": "https",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   "TCP",
						Port:       80,
						TargetPort: intstr.FromInt(555),
						NodePort:   31080,
					},
					{
						Name:       "https",
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(555),
						NodePort:   31443,
					},
				},
				Type: "NodePort",
			},
		}

		hostCluster = &clustercrd.Cluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-host-cluster",
				Namespace: "nautes",
				Annotations: map[string]string{
					"ver": "1",
				},
			},
			Spec: clustercrd.ClusterSpec{
				ApiServer:   "https://127.0.0.1",
				ClusterType: clustercrd.CLUSTER_TYPE_PHYSICAL,
				ClusterKind: clustercrd.CLUSTER_KIND_KUBERNETES,
				Usage:       clustercrd.CLUSTER_USAGE_HOST,
				HostCluster: "",
			},
		}

		cluster = &clustercrd.Cluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-cluster",
				Namespace: "nautes",
				Annotations: map[string]string{
					"ver": "1",
				},
			},
			Spec: clustercrd.ClusterSpec{
				ApiServer:   "https://127.0.0.1:8081",
				ClusterType: clustercrd.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: clustercrd.CLUSTER_KIND_KUBERNETES,
				Usage:       clustercrd.CLUSTER_USAGE_WORKER,
				HostCluster: "test-host-cluster",
			},
		}

		wantErr = nil
		wantResult = &secretclient.SyncResult{
			SecretID: "1",
		}
	})

	AfterEach(func() {
		wantErr = nil
		err := k8sClient.Delete(context.Background(), svc)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = k8sClient.Delete(context.Background(), hostCluster)
		Expect(client.IgnoreNotFound(err)).Should(BeNil())
		err = WaitForDelete(cluster)
		Expect(err).Should(BeNil())
		err = WaitForDelete(hostCluster)
		Expect(err).Should(BeNil())
	})

	It("when entrypoint is nodeport, get entrypoint nodeport from hostcluster", func() {
		err := k8sClient.Create(ctx, svc)
		Expect(err).Should(BeNil())

		startTime := time.Now()
		time.Sleep(time.Second)
		err = k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("True"))

		syncedCluster := &clustercrd.Cluster{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, syncedCluster)
		Expect(err).Should(BeNil())
		Expect(syncedCluster.Status.EntryPoints).ShouldNot(BeNil())
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPPort).Should(Equal(svc.Spec.Ports[0].NodePort))
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPSPort).Should(Equal(svc.Spec.Ports[1].NodePort))
		Expect(string(syncedCluster.Status.EntryPoints[svc.Name].Type)).Should(Equal(string(svc.Spec.Type)))
	})

	It("when entrypoint is loadbalancer, get entrypoint port from hostcluster", func() {
		svc.Spec.Type = "LoadBalancer"
		err := k8sClient.Create(ctx, svc)
		Expect(err).Should(BeNil())

		startTime := time.Now()
		time.Sleep(time.Second)
		err = k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("True"))

		syncedCluster := &clustercrd.Cluster{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, syncedCluster)
		Expect(err).Should(BeNil())
		Expect(syncedCluster.Status.EntryPoints).ShouldNot(BeNil())
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPPort).Should(Equal(svc.Spec.Ports[0].Port))
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPSPort).Should(Equal(svc.Spec.Ports[1].Port))
		Expect(string(syncedCluster.Status.EntryPoints[svc.Name].Type)).Should(Equal(string(svc.Spec.Type)))
	})

	It("if svc only mark http, cluster entrypoint will only record http", func() {
		svc.Annotations = map[string]string{
			"cluster-operator.nautes.io/http-entrypoint": "http",
		}
		err := k8sClient.Create(ctx, svc)
		Expect(err).Should(BeNil())

		startTime := time.Now()
		time.Sleep(time.Second)
		err = k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("True"))

		syncedCluster := &clustercrd.Cluster{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, syncedCluster)
		Expect(err).Should(BeNil())
		Expect(syncedCluster.Status.EntryPoints).ShouldNot(BeNil())
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPPort).Should(Equal(svc.Spec.Ports[0].NodePort))
		Expect(syncedCluster.Status.EntryPoints[svc.Name].HTTPSPort).Should(Equal(int32(0)))
	})

	It("if svc not match annotation, cluster will not record entrypoint", func() {
		svc.Annotations = map[string]string{
			"cluster-operator.nautes.io/http-entrypoint": "httpt",
		}
		err := k8sClient.Create(ctx, svc)
		Expect(err).Should(BeNil())

		startTime := time.Now()
		time.Sleep(time.Second)
		err = k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))

		syncedCluster := &clustercrd.Cluster{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, syncedCluster)
		Expect(err).Should(BeNil())
		Expect(syncedCluster.Status.EntryPoints).Should(BeNil())
	})

	It("if svc type is not support, cluster will not record entrypoint", func() {
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Ports[0].NodePort = 0
		svc.Spec.Ports[1].NodePort = 0
		err := k8sClient.Create(ctx, svc)
		Expect(err).Should(BeNil())

		startTime := time.Now()
		time.Sleep(time.Second)
		err = k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))

		syncedCluster := &clustercrd.Cluster{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}, syncedCluster)
		Expect(err).Should(BeNil())
		Expect(syncedCluster.Status.EntryPoints).Should(BeNil())
	})

	It("when collect entrypoint failed, record error message.", func() {
		wantErr = errors.New("this is error")

		startTime := time.Now()
		time.Sleep(time.Second)
		err := k8sClient.Create(context.Background(), hostCluster)
		Expect(err).Should(BeNil())
		err = k8sClient.Create(context.Background(), cluster)
		Expect(err).Should(BeNil())
		cdt, err := WaitForCondition(cluster, startTime, clusterConditionTypeEntryPointCollection)
		Expect(err).Should(BeNil())
		Expect(string(cdt.Status)).Should(Equal("False"))
		Expect(cdt.Message).Should(ContainSubstring(wantErr.Error()))
	})
})
