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

package coderepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"

	utilstrings "github.com/nautes-labs/nautes/app/argo-operator/util/strings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errNotFoundCodeRepo = errors.New("coderepo is not found")
	errCreateRepository = errors.New("failed to create repository")
	errDeleteRepository = errors.New("failed to delete repository")
	errGetSecret        = errors.New("failed to get secret")
)

func generateResourceName() string {
	return fmt.Sprintf("coderepo-%s", utilstrings.RandStringRunes(5))
}

var _ = Describe("CodeRepo controller test cases", func() {
	const (
		timeout           = time.Second * 20
		interval          = time.Second * 5
		codeRepoURL       = "git@158.222.222.235:nautes/test.git"
		codeRepoUpdateURL = "ssh://git@gitlab.com:2222/nautes-labs/test.git"
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
		codeRepoReponse = &argocd.CodeRepoResponse{
			Name: "argo-operator-test-vcluster",
			Repo: codeRepoURL,
			ConnectionState: argocd.ConnectionState{
				Status: "Successful",
			},
		}
		codeRepoFailReponse = &argocd.CodeRepoResponse{
			Name: "argo-operator-test-vcluster",
			Repo: codeRepoURL,
			ConnectionState: argocd.ConnectionState{
				Status: "Fail",
			},
		}
		errNotFound = &argocd.ErrorNotFound{Code: 404, Message: errNotFoundCodeRepo.Error()}
	)

	It("successfully create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expected resource created successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(BeNil())

		fakeCtl.close()
	})

	It("successfully update repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()
		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected resource created successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			err := k8sClient.Delete(context.Background(), codeRepo)
			return err
		}, timeout, interval).Should(BeNil())

		fakeCtl.close()
	})

	It("is resource validate error", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, nil, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected resource created successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			err := k8sClient.Delete(context.Background(), codeRepo)
			return err
		}, timeout, interval).Should(BeNil())

		By("Expected resource deletion finished")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			return err
		}, timeout, interval).ShouldNot(Succeed())

		fakeCtl.close()
	})

	It("updates repository to argocd when secret has change", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		firstGetSecret := sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   true,
			DeploymentRuntime: false,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"TagPushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if codeRepo.Status.Sync2ArgoStatus != nil {
				return codeRepo.Status.Sync2ArgoStatus.SecretID == "2"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("creates repository to argocd when secret has change", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		secondGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(secondGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		firstGetSecret := sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting resource added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   true,
			DeploymentRuntime: false,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"TagPushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if codeRepo.Status.Sync2ArgoStatus != nil {
				return codeRepo.Status.Sync2ArgoStatus.SecretID == "2"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("modify resource successfully and update repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			conditions := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})
			return len(conditions) > 0
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoUpdateURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, toUpdate)
			toUpdate.Spec = updateSpec
			err = k8sClient.Update(context.Background(), toUpdate)
			return err
		})
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())

			conditions := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})

			return len(conditions) > 0
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("modify resource successfully and create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 3)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the coderepo is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoUpdateURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, toUpdate)
			toUpdate.Spec = updateSpec
			err = k8sClient.Update(context.Background(), toUpdate)
			return err
		})
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())
			conditions := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(errCreateRepository).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected conditions of resource status is nil")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to get secret data", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(errCreateRepository).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{}, errGetSecret).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expected conditions of resource status is nil")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to delete repository of argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(errDeleteRepository).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().Init(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName()
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               codeRepoURL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			coderepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, coderepo)
			return coderepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion failed")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			if err != nil {
				return err
			}
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		By("Expected resource deletion incomplete")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			return k8sClient.Get(context.Background(), key, codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})
})
