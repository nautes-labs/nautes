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

package biz

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	clusterregistration "github.com/nautes-labs/nautes/app/api-server/pkg/cluster"
	"github.com/nautes-labs/nautes/app/api-server/pkg/gitlab"
	"github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	clusterConfig "github.com/nautes-labs/nautes/pkg/config/cluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Save cluster", func() {
	var (
		tenantRepositoryHttpsURL  = fmt.Sprintf("%s/dev-test-tenant/management.git", nautesConfigs.Git.Addr)
		clusterTemplateLocalPath  = "/tmp/product/cluster-templates"
		tenantRepositoryLocalPath = "/tmp/product/cluster-templates"
		secretPath                = "default"
		cacertSecretOptions       = &SecretOptions{
			SecretPath:   secretPath,
			SecretEngine: "pki",
			SecretKey:    "cacert",
		}
		virtualCluster = &resourcev1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: resourcev1alpha1.GroupVersion.String(),
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "virtual-cluster1",
				Namespace: nautesConfigs.Nautes.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   "https://kubernetes.svc",
				ClusterType: resourcev1alpha1.ClusterType("virtual"),
				ClusterKind: resourcev1alpha1.ClusterKind("kubernetes"),
				Usage:       resourcev1alpha1.ClusterUsage("worker"),
				HostCluster: "host216",
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
		physcialCluster = &resourcev1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: resourcev1alpha1.GroupVersion.String(),
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "physical-cluster1",
				Namespace: nautesConfigs.Nautes.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   "https://kubernetes.svc",
				ClusterType: resourcev1alpha1.ClusterType("physical"),
				ClusterKind: resourcev1alpha1.ClusterKind("kubernetes"),
				Usage:       resourcev1alpha1.ClusterUsage("worker"),
				WorkerType:  resourcev1alpha1.ClusterWorkTypeDeployment,
				ComponentsList: resourcev1alpha1.ComponentsList{
					CertManagement: &resourcev1alpha1.Component{Name: "cert-manager", Namespace: "cert-manager"},
					Deployment:     &resourcev1alpha1.Component{Name: "argocd", Namespace: "argocd"},
					EventListener:  &resourcev1alpha1.Component{Name: "argo-events", Namespace: "argo-events"},
					Gateway: &resourcev1alpha1.Component{
						Name:      "traefik",
						Namespace: "traefik",
						Additions: map[string]string{
							"httpsNodePort": "30020",
							"httpNodePort":  "30221",
						},
					},
					MultiTenant:         &resourcev1alpha1.Component{Name: "hnc", Namespace: "hnc"},
					Pipeline:            &resourcev1alpha1.Component{Name: "tekton", Namespace: "tekton-pipelines"},
					ProgressiveDelivery: &resourcev1alpha1.Component{Name: "argo-rollouts", Namespace: "argo-rollouts"},
					SecretManagement:    &resourcev1alpha1.Component{Name: "vault", Namespace: "vault"},
					SecretSync:          &resourcev1alpha1.Component{Name: "external-secrets", Namespace: "external-secrets"},
					OauthProxy:          &resourcev1alpha1.Component{Name: "oauth2-proxy", Namespace: "oauth2-proxy"},
				},
			},
		}
		hostCluster = &resourcev1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: resourcev1alpha1.GroupVersion.String(),
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "host-cluster1",
				Namespace: nautesConfigs.Nautes.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   "https://kubernetes.svc",
				ClusterType: resourcev1alpha1.ClusterType("physical"),
				ClusterKind: resourcev1alpha1.ClusterKind("kubernetes"),
				Usage:       resourcev1alpha1.ClusterUsage("host"),
				ComponentsList: resourcev1alpha1.ComponentsList{
					Gateway: &resourcev1alpha1.Component{
						Name:      "traefik",
						Namespace: "traefik",
						Additions: map[string]string{
							"httpsNodePort": "30020",
							"httpNodePort":  "30221",
						},
					},
					OauthProxy: &resourcev1alpha1.Component{
						Name:      "oauth2-proxy",
						Namespace: "oauth2-proxy",
					},
				},
			},
		}
		clusterTemplateCloneParam = &CloneRepositoryParam{
			URL:   nautesConfigs.Nautes.RuntimeTemplateSource,
			User:  _GitUser,
			Email: _GitEmail,
		}
		tenantConfigCloneParam = &CloneRepositoryParam{
			URL:   tenantRepositoryHttpsURL,
			User:  _GitUser,
			Email: _GitEmail,
		}
		param = &clusterregistration.ClusterRegistrationParam{
			Traefik: &clusterregistration.Traefik{
				HttpNodePort:  "30012",
				HttpsNodePort: "30013",
			},
			ArgocdHost: "https://argocd.com",
		}
		tenant = &Project{
			ID:                int32(22),
			Name:              "repo-22",
			Path:              "repo-22",
			SshUrlToRepo:      tenantRepositoryHttpsURL,
			HttpUrlToRepo:     tenantRepositoryHttpsURL,
			PathWithNamespace: fmt.Sprintf("%v/%v", defaultProductGroup.Path, defaultProjectName),
		}
		argocdOAuthURL = "argocd.test.10.231.322.312.nip.io/api/dex/callback"
		tektonOAuthURL = "auth.test.10.231.322.312.nip.io/oauth/callback"
		kubeconfig     = "apiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdGMyVnkKZG1WeUxXTmhRREUyTlRVM01EVTFOemt3SGhjTk1qSXdOakl3TURZeE1qVTVXaGNOTXpJd05qRTNNRFl4TWpVNQpXakFqTVNFd0h3WURWUVFEREJock0zTXRjMlZ5ZG1WeUxXTmhRREUyTlRVM01EVTFOemt3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFUeXhrT2xIVm52ZkJlN01MUUU2N0ZpTTBTcDY2eXREYi8ydUY3MWdWclAKeFk2cDlZRm5YWU5PYXA2bktZZ2hMMzRFVU9FUHNFTWh0YTZ3bGxQWnNXeG5vMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVUZHaGY4czJwYkFjT0svL3ZrdU5GCnVMS3d6U0V3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnU2dRUnplKzE2TG40aWRvbUg5Zk40bjc4dE9VSHFMdlgKL2V5RjFpL1NwQ2tDSVFDNy9Fcmc2UUtXSjRRQXl5QlZuSFloOVNvd3FiZWkwSk83c2tFY01zWVhkZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    server: https://127.0.0.1:6443\n  name: default\ncontexts:\n- context:\n    cluster: default\n    user: default\n  name: default\ncurrent-context: default\nkind: Config\npreferences: {}\nusers:\n- name: default\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrakNDQVRlZ0F3SUJBZ0lJRDRCbThDSk5sM2N3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOalUxTnpBMU5UYzVNQjRYRFRJeU1EWXlNREEyTVRJMU9Wb1hEVEl6TURZeQpNREEyTVRJMU9Wb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJGYXhQSFpjb0tLRDJmRWsKZHlsYm9GNXNPZzNQRE85ZGRqdGt2cGdRTEdtaEw0RUlUQUZWRlRVYndJV0l0aUdTa0RQQVhzVjVibDFtYXhhawpvVXFJcEhXalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCUnpEZ3AxMERYeEhtL3F4M0lOenRxMlJORnUxekFLQmdncWhrak9QUVFEQWdOSkFEQkcKQWlFQXRzVVVFRkIwRXY3b1IwRmtvT2JSVE1NM25oQVNNZFMwMHQvOGdYNmRybDhDSVFEbGIvSTNNelNnV2JodQpNR0Fqc2l4WlYrRitkWkF6emNJSzg2ZjFWZXRBdUE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCi0tLS0tQkVHSU4gQ0VSVElGSUNBVEUtLS0tLQpNSUlCZHpDQ0FSMmdBd0lCQWdJQkFEQUtCZ2dxaGtqT1BRUURBakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwClpXNTBMV05oUURFMk5UVTNNRFUxTnprd0hoY05Nakl3TmpJd01EWXhNalU1V2hjTk16SXdOakUzTURZeE1qVTUKV2pBak1TRXdId1lEVlFRRERCaHJNM010WTJ4cFpXNTBMV05oUURFMk5UVTNNRFUxTnprd1dUQVRCZ2NxaGtqTwpQUUlCQmdncWhrak9QUU1CQndOQ0FBVHpKZC92QTE1NCtmVkRSUFlZNzZqTVh5OHJzZmhMUEZrSElnUERCQm9nClJYKzNlNlFOQStHV3pNemtDQWpjUVBJMVdaWHM0YWU3WlB4MkhnWVpGVmJDbzBJd1FEQU9CZ05WSFE4QkFmOEUKQkFNQ0FxUXdEd1lEVlIwVEFRSC9CQVV3QXdFQi96QWRCZ05WSFE0RUZnUVVjdzRLZGRBMThSNXY2c2R5RGM3YQp0a1RSYnRjd0NnWUlLb1pJemowRUF3SURTQUF3UlFJaEFON2JjOTFiSEpvYXhiUWhOVDZLS0F6dE1IdzM5OHhqCkJsTks1Wm9lY0VYc0FpQUx1TWZJUWNwUjI4ekE4d2g1YTdheVR3SSt5bTJ0enliQitSZnJkQVJqb1E9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==\n    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSU56Rk9hWXJjaHRqL3kyV1p5UFJia29UQjhTVmNaM1JZSnN3OGs4eUJHMEtvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFVnJFOGRseWdvb1BaOFNSM0tWdWdYbXc2RGM4TTcxMTJPMlMrbUJBc2FhRXZnUWhNQVZVVgpOUnZBaFlpMklaS1FNOEJleFhsdVhXWnJGcVNoU29pa2RRPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=\n"
	)

	u, err := url.Parse(nautesConfigs.Git.Addr)
	Expect(err).ShouldNot(HaveOccurred())
	gitlabCertPath := fmt.Sprintf("%s/%s.crt", gitlab.SSLDirectory, u.Hostname())

	BeforeEach(func() {
		dir := filepath.Dir(gitlabCertPath)
		err := os.MkdirAll(dir, os.ModePerm)
		Expect(err).ShouldNot(HaveOccurred())

		_, err = os.Create(gitlabCertPath)
		Expect(err).ShouldNot(HaveOccurred())

		err = clusterConfig.SetClusterValidateConfig()
		Expect(err).Should(BeNil())

	})

	AfterEach(func() {
		err = os.RemoveAll(gitlabCertPath)
		Expect(err).ShouldNot(HaveOccurred())

		err = clusterConfig.DeleteValidateConfig()
		Expect(err).Should(BeNil())
	})

	It("successfully saved host cluster", func() {
		param.Cluster = hostCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetSecret(gomock.Any(), cacertSecretOptions).Return("cacert", nil)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), tenantRepositoryLocalPath)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().GetTektonOAuthURL().Return(tektonOAuthURL, nil)
		clusteroperator.EXPECT().Save().Return(nil)

		dex := NewMockDexRepo(ctl)
		dex.EXPECT().UpdateRedirectURIs(tektonOAuthURL).Return(nil)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to saved kubeconfig", func() {
		param.Cluster = hostCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(errors.New("failed to saved kubeconfig"))

		gitRepo := NewMockGitRepo(ctl)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to clone cluster template", func() {
		param.Cluster = hostCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return("", errors.New("failed to get cluster template repository"))

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)

		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to clone tenant config repository", func() {
		param.Cluster = hostCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, errors.New("failed to get trnant config repository"))

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)
		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)

		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).Should(HaveOccurred())
	})

	It("successfully saved physical runtime", func() {
		param.Cluster = physcialCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetSecret(gomock.Any(), cacertSecretOptions).Return("cacert", nil)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), tenantRepositoryLocalPath)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().Save().Return(nil)
		clusteroperator.EXPECT().GetArgocdURL().Return(argocdOAuthURL, nil)
		clusteroperator.EXPECT().GetTektonOAuthURL().Return(tektonOAuthURL, nil)

		dex := NewMockDexRepo(ctl)
		dex.EXPECT().UpdateRedirectURIs(argocdOAuthURL).Return(nil)
		dex.EXPECT().UpdateRedirectURIs(tektonOAuthURL).Return(nil)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("successfully saved virtual runtime", func() {
		param.Cluster = virtualCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetSecret(gomock.Any(), cacertSecretOptions).Return("cacert", nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), tenantRepositoryLocalPath)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().Save().Return(nil)
		clusteroperator.EXPECT().GetArgocdURL().Return(argocdOAuthURL, nil)

		dex := NewMockDexRepo(ctl)
		dex.EXPECT().UpdateRedirectURIs(argocdOAuthURL).Return(nil)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to get tenant repository", func() {
		param.Cluster = virtualCluster

		client := kubernetes.NewMockClient(ctl)
		firstList := client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("codeRepo is not found")).After(firstList)

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil).AnyTimes()

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)

		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to saved cluster", func() {
		param.Cluster = physcialCluster

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)
		secretRepo.EXPECT().GetSecret(gomock.Any(), cacertSecretOptions).Return("cacert", nil)
		secretRepo.EXPECT().SaveClusterConfig(gomock.Any(), param.Cluster.Name, gomock.Any()).Return(nil)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().Save().Return(fmt.Errorf("failed save cluster"))

		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.SaveCluster(context.Background(), param, kubeconfig)
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("Delete cluster", func() {
	var (
		tenantRepositoryHttpsURL  = fmt.Sprintf("%s/dev-test-tenant/management.git", nautesConfigs.Git.Addr)
		clusterTemplateLocalPath  = "/tmp/product/cluster-templates"
		tenantRepositoryLocalPath = "/tmp/product/cluster-templates"
		cluster                   = &resourcev1alpha1.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: resourcev1alpha1.GroupVersion.String(),
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "virtual-cluster1",
				Namespace: nautesConfigs.Nautes.Namespace,
			},
			Spec: resourcev1alpha1.ClusterSpec{
				ApiServer:   "https://kubernetes.svc",
				ClusterType: resourcev1alpha1.ClusterType("virtual"),
				ClusterKind: resourcev1alpha1.ClusterKind("kubernetes"),
				Usage:       resourcev1alpha1.ClusterUsage("worker"),
			},
		}
		clusterFilePath           = fmt.Sprintf("%s/nautes/overlays/production/clusters/%s.yaml", tenantRepositoryLocalPath, cluster.Name)
		clusterTemplateCloneParam = &CloneRepositoryParam{
			URL:   nautesConfigs.Nautes.RuntimeTemplateSource,
			User:  _GitUser,
			Email: _GitEmail,
		}
		tenantConfigCloneParam = &CloneRepositoryParam{
			URL:   tenantRepositoryHttpsURL,
			User:  _GitUser,
			Email: _GitEmail,
		}
		tenant = &Project{
			ID:                int32(22),
			Name:              "repo-22",
			Path:              "repo-22",
			SshUrlToRepo:      tenantRepositoryHttpsURL,
			HttpUrlToRepo:     tenantRepositoryHttpsURL,
			PathWithNamespace: fmt.Sprintf("%v/%v", defaultProductGroup.Path, defaultProjectName),
		}
		argocdOAuthURL = "argocd.test.10.231.322.312.nip.io"
	)

	It("successfully deleted cluster", func() {
		createFileIfNotExist(clusterFilePath)
		defer deleteFileIfExists(clusterFilePath)

		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)
		gitRepo.EXPECT().Fetch(gomock.Any(), gomock.Any(), "origin").Return("any", nil)
		gitRepo.EXPECT().Diff(gomock.Any(), gomock.Any(), "main", "remotes/origin/main").Return("", nil)
		gitRepo.EXPECT().SaveConfig(gomock.Any(), tenantRepositoryLocalPath)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().GetArgocdURL().Return(argocdOAuthURL, nil)
		clusteroperator.EXPECT().Remove().Return(nil)

		dex := NewMockDexRepo(ctl)
		dex.EXPECT().RemoveRedirectURIs(argocdOAuthURL).Return(nil)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.DeleteCluster(context.Background(), cluster.Name)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("failed to deleted cluster", func() {
		createFileIfNotExist(clusterFilePath)
		defer deleteFileIfExists(clusterFilePath)
		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)
		clusteroperator.EXPECT().InitializeClusterConfig(gomock.Any()).Return(nil)
		clusteroperator.EXPECT().Remove().Return(fmt.Errorf("failed to delete cluster"))
		clusteroperator.EXPECT().GetArgocdURL().Return(argocdOAuthURL, nil)

		dex := NewMockDexRepo(ctl)
		dex.EXPECT().RemoveRedirectURIs(argocdOAuthURL).Return(nil)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.DeleteCluster(context.Background(), cluster.Name)
		Expect(err).Should(HaveOccurred())
	})

	It("failed to get cluster", func() {
		client := kubernetes.NewMockClient(ctl)
		client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

		codeRepo := NewMockCodeRepo(ctl)
		codeRepo.EXPECT().GetCurrentUser(gomock.Any()).Return(_GitUser, _GitEmail, nil).AnyTimes()
		codeRepo.EXPECT().GetCodeRepo(gomock.Any(), gomock.Any()).Return(tenant, nil)

		secretRepo := NewMockSecretrepo(ctl)

		gitRepo := NewMockGitRepo(ctl)
		gitRepo.EXPECT().Clone(gomock.Any(), clusterTemplateCloneParam).Return(clusterTemplateLocalPath, nil)
		gitRepo.EXPECT().Clone(gomock.Any(), tenantConfigCloneParam).Return(tenantRepositoryLocalPath, nil)

		resourceusecase := NewResourcesUsecase(logger, codeRepo, secretRepo, gitRepo, nil, nautesConfigs)

		clusteroperator := clusterregistration.NewMockClusterRegistrationOperator(ctl)

		dex := NewMockDexRepo(ctl)

		clusterusecase := NewClusterUsecase(logger, codeRepo, secretRepo, resourceusecase, nautesConfigs, client, clusteroperator, dex)
		err := clusterusecase.DeleteCluster(context.Background(), cluster.Name)
		Expect(err).Should(HaveOccurred())
	})
})

// Check if file exists and create if it does not exist
func createFileIfNotExist(filename string) (*os.File, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		dir := filepath.Dir(filename)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		}

		if file, err := os.Create(filename); err != nil {
			return nil, err
		} else {
			return file, nil
		}
	} else if err != nil {

		return nil, err
	} else {

		return nil, nil
	}
}

// Check if file exists and delete if it exists
func deleteFileIfExists(filename string) error {

	fileInfo, err := os.Stat(filename)
	if os.IsNotExist(err) {

		return nil
	} else if err != nil {

		return err
	}

	if fileInfo.IsDir() {

		err = filepath.Walk(filename, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			return os.Remove(path)
		})
		if err != nil {
			return err
		}

		err = os.Remove(filename)
		if err != nil {
			return err
		}
	} else {
		err = os.Remove(filename)
		if err != nil {
			return err
		}
	}

	return nil
}
