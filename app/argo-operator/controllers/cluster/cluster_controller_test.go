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
	"errors"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	pkgsecret "github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utilstrings "github.com/nautes-labs/nautes/app/argo-operator/util/strings"
	clusterConfig "github.com/nautes-labs/nautes/pkg/config/cluster"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

var (
	errSyncCluster     = errors.New("cluster addition failed")
	errDeleteCluster   = errors.New("cluster deletion failed")
	errNotFoundCluster = errors.New("cluster is not found")
	errGetSecret       = errors.New("secret path is error")
)

func spliceResourceName() string {
	return fmt.Sprintf("cluster-%s", utilstrings.RandStringRunes(5))
}

var _ = Describe("Cluster controller test cases", func() {
	const (
		timeout          = time.Second * 20
		interval         = time.Second * 3
		k8sDefaultServer = "https://kubernetes.default.svc"
		k8sTestServer    = "https://kubernetes.test.svc"
	)

	var (
		k8sClient    client.Client
		fakeCtl      *fakeController
		nautesConfig = &nautesconfigs.Config{
			Git: nautesconfigs.GitRepo{
				Addr:    "https://gitlab.bluzin.io",
				GitType: "gitlab",
			},
			Secret: nautesconfigs.SecretRepo{
				RepoType: "vault",
				Vault: nautesconfigs.Vault{
					Addr:      "https://vault.bluzin.io:8200",
					MountPath: "deployment2-runtime",
					ProxyAddr: "https://vault.bluzin.io:8000",
					Token:     "hvs.UseDJnYBNykFeGWlh7YAnzjC",
				},
			},
		}
		secretData = "apiVersion: v1\nclusters:\n- cluster:\n    server: https://127.0.0.1:6443\n    insecure-skip-tls-verify: true\n  name: local\ncontexts:\n- context:\n    cluster: local\n    namespace: default\n    user: user\n  name: pipeline1-runtime\ncurrent-context: pipeline1-runtime\nkind: Config\npreferences: {}\nusers:\n- name: user\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrRENDQVRlZ0F3SUJBZ0lJS005ZXhoLzlncXd3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOalUyTXprNU5UYzFNQjRYRFRJeU1EWXlPREEyTlRrek5Wb1hEVEl6TURZeQpPREEyTlRrek5Wb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJMNEJ3TDFBMVhYaStTcm8KYlJRSklFS3FhMlUycEZKV2NUcWJ1SmtKdVp5ZDBrdFBUOHhscmVoQ1phUzhRajFHdVVvWmNBM1A5eE12dVBWTgovTUx6UGVXalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCU3VTbUJQeFRmdHVoSXBHczBpSjRSd0MyVVY5VEFLQmdncWhrak9QUVFEQWdOSEFEQkUKQWlCckF5S2lNaWZUTUs3MThpd0hkVGZxUFI1TU96QW9OTjVvajZDak9uSzZtUUlnZG5HNFJQVjJtYTQ4R2FmQQovVXJUeEV3R0g3WVcwQ3VvUzZjL0tjWU5RbjA9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUyTlRZek9UazFOelV3SGhjTk1qSXdOakk0TURZMU9UTTFXaGNOTXpJd05qSTFNRFkxT1RNMQpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUyTlRZek9UazFOelV3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFROFdnVHJ6cDNkZVNlMFlVTTNIamhQZGU2VWRoNGN2L0h2RStTci9BcDQKVGxnc2srSVcwSG5jSHoyVXQ2c0lXMkRRenNVMnJyMGZXZ0pmVlNZendoS0NvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXJrcGdUOFUzN2JvU0tSck5JaWVFCmNBdGxGZlV3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnSGJ3aFdLQ2Jxa3IxcEFRdk04bGtrNC9Pc0hiWVZZTEMKVVo5Q1lmVWx1bE1DSVFEcW9TVVBkb3F0cUpTRnZ2bnkxQjBVeXBkRWRzemMzczBuSk5PekhEdU12dz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSVBKZmQvVEx0MnZSczJEOHlWVklTV0xSL3NHVm1ZbjRvNjFMVkNMb29VZUtvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFdmdIQXZVRFZkZUw1S3VodEZBa2dRcXByWlRha1VsWnhPcHU0bVFtNW5KM1NTMDlQekdXdAo2RUpscEx4Q1BVYTVTaGx3RGMvM0V5KzQ5VTM4d3ZNOTVRPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=\n"
	)

	var (
		clusterReponse = &argocd.ClusterResponse{
			Name:   "argo-operator-test-vcluster",
			Server: k8sDefaultServer,
			ConnectionState: argocd.ConnectionState{
				Status: "Successful",
			},
		}
		clusterFailReponse = &argocd.ClusterResponse{
			Name:   "argo-operator-test-vcluster",
			Server: k8sDefaultServer,
			ConnectionState: argocd.ConnectionState{
				Status: "Fail",
			},
		}
	)

	BeforeEach(func() {
		err := clusterConfig.SetClusterValidateConfig()
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		err := clusterConfig.DeleteValidateConfig()
		Expect(err).Should(BeNil())
	})

	It("successfully create cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		first := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).After(first)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: DefaultNamespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err := k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to update cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterFailReponse, nil).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().UpdateCluster(gomock.Any()).Return(errors.New("failed to update cluster")).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   k8sDefaultServer,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}
		toUpdate := toCreate.DeepCopy()
		toUpdate.Spec = updateSpec

		By("Expected resource update successfully")
		k8sClient.Get(context.Background(), key, toUpdate)
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "False"
			}

			return false
		}, timeout*2, interval*2).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("cluster validate error", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, nil, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_PHYSICAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("when secret has change create cluster to argocd", func() {
		var err error
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		secondGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes().Times(2)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(secondGetCluster)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sTestServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		firstGetSecret := secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil)
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   k8sDefaultServer,
			ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
			ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
			HostCluster: "tenant1",
			Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
			ComponentsList: resourcev1alpha1.ComponentsList{
				CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
				Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
				EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
				MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
				Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
				ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
				SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
				SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
				OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
			},
		}

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 6)

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   k8sDefaultServer,
			ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
			ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
			HostCluster: "tenant2",
			Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
			WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
			ComponentsList: resourcev1alpha1.ComponentsList{
				CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
				Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
				EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
				MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
				Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
				ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
				SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
				SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
				OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
			},
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			if cluster.Status.Sync2ArgoStatus != nil {
				return cluster.Status.Sync2ArgoStatus.SecretID == "2"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err := k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("update resource successfully update cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		secondGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes().Times(2)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(secondGetCluster)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().UpdateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sTestServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   k8sTestServer,
			ClusterKind: "kubernetes",
			ClusterType: "virtual",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("update resource successfully create cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetClusterInfo := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).After(firstGetClusterInfo).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sTestServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()

		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   k8sTestServer,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to save cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		first := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).After(first)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()

		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expected conditions of resource status is nil")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil && len(cluster.Status.Conditions) > 0 {
				return cluster.Status.Conditions[0].Status == "False"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to get secret data", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()

		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{}, errGetSecret).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expected conditions of resource status is nil")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Status.Sync2ArgoStatus == nil
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		defer fakeCtl.close()
	})

	It("failed to create cluster to argocd", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetClusterInfo := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).After(firstGetClusterInfo).AnyTimes()
		firstCreateCluster := argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).After(firstCreateCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()

		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}
		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err := k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc2",
			ClusterKind: "kubernetes",
			ClusterType: "virtual",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to save cluster to argocd while updating resources", func() {
		var err error

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		secondGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).Times(2)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(secondGetCluster)
		firstCreateCluster := argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).After(firstCreateCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(k8sDefaultServer, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()

		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		var resourceName = spliceResourceName()
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   k8sDefaultServer,
				ClusterType: resourcev1alpha1.CLUSTER_TYPE_VIRTUAL,
				ClusterKind: resourcev1alpha1.CLUSTER_KIND_KUBERNETES,
				HostCluster: "tenant1",
				Usage:       resourcev1alpha1.CLUSTER_USAGE_WORKER,
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement:      &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:          &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:       &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc2",
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}
		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec

		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		cluster := &resourcev1alpha1.Cluster{}
		err = k8sClient.Get(context.Background(), key, cluster)
		Expect(err).ShouldNot(HaveOccurred())
		err = k8sClient.Delete(context.Background(), cluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err != nil
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})
})
