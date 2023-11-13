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

package argocd_test

import (
	"context"
	"fmt"
	"strings"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployment/argocd"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rbacKey = "policy.csv"
)

var _ = Describe("ArgoCD", func() {
	var err error
	var ctx context.Context
	var productID string
	var appNames []string
	var repoNames []string
	var spaceNames []string
	var userNames []string
	var seed string
	var deployer syncer.Deployment
	var providerName = "provider"
	var db *database.RuntimeDataBase

	BeforeEach(func() {
		ctx = context.TODO()
		seed = RandNum()
		productID = fmt.Sprintf("product-%s", seed)
		productName := fmt.Sprintf("productName-%s", seed)

		appNames = GenerateNames(fmt.Sprintf("app-%s-%%d", seed), 5)
		repoNames = GenerateNames(fmt.Sprintf("repo-%s-%%d", seed), 5)
		spaceNames = GenerateNames(fmt.Sprintf("space-%s-%%d", seed), 5)
		userNames = GenerateNames(fmt.Sprintf("user-%s-%%d", seed), 5)
		opts := v1alpha1.Component{
			Name:      "",
			Namespace: argoCDNamespace,
			Additions: nil,
		}

		clusterName := fmt.Sprintf("cluster-%s", seed)

		db = &database.RuntimeDataBase{
			Clusters: map[string]v1alpha1.Cluster{
				clusterName: {
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterName,
						Namespace: nautesNamespace,
					},
					Spec: v1alpha1.ClusterSpec{
						ReservedNamespacesAllowedProducts: map[string][]string{
							"ns1": {productName},
						},
						ProductAllowedClusterResources: map[string][]v1alpha1.ClusterResourceInfo{
							productName: {v1alpha1.ClusterResourceInfo{
								Kind:  "*",
								Group: "*",
							}},
						},
					},
					Status: v1alpha1.ClusterStatus{
						ProductIDMap: map[string]string{
							productName: productID,
						},
					},
				},
			},
			CodeRepoProviders: map[string]v1alpha1.CodeRepoProvider{
				providerName: {
					ObjectMeta: metav1.ObjectMeta{
						Name:      providerName,
						Namespace: nautesNamespace,
					},
					Spec: v1alpha1.CodeRepoProviderSpec{
						ProviderType: "gitlab",
					},
				},
			},
			CodeRepos: map[string]v1alpha1.CodeRepo{
				repoNames[0]: {
					ObjectMeta: metav1.ObjectMeta{
						Name: repoNames[0],
					},
					Spec: v1alpha1.CodeRepoSpec{
						CodeRepoProvider: providerName,
						Product:          productID,
						Project:          "",
						RepoName:         "repoNameTest",
						URL:              "ssh://127.0.0.1/test/repoNameTest.git",
					},
				},
				repoNames[1]: {
					ObjectMeta: metav1.ObjectMeta{
						Name: repoNames[1],
					},
					Spec: v1alpha1.CodeRepoSpec{
						CodeRepoProvider: providerName,
						Product:          productID,
						Project:          "",
						RepoName:         "repoNameTest",
						URL:              "ssh://127.0.0.1/test/repoNameTest.git",
					},
				},
				repoNames[2]: {
					ObjectMeta: metav1.ObjectMeta{
						Name: repoNames[2],
					},
					Spec: v1alpha1.CodeRepoSpec{
						CodeRepoProvider: providerName,
						Product:          productID,
						Project:          "",
						RepoName:         "repoNameTest",
						URL:              "ssh://127.0.0.1/test/repoNameTest.git",
					},
				},
			},
		}

		initInfo := syncer.ComponentInitInfo{
			ClusterConnectInfo: syncer.ClusterConnectInfo{
				ClusterKind: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &syncer.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName:            clusterName,
			NautesResourceSnapshot: db,
			NautesConfig: configs.Config{
				Nautes: configs.Nautes{
					Namespace: nautesNamespace,
					ServiceAccount: map[string]string{
						configs.OperatorNameArgo: "argo",
					},
				},
				Secret: configs.SecretRepo{
					OperatorName: map[string]string{
						configs.OperatorNameArgo: "argo",
					},
				},
			},
			Components: &syncer.ComponentList{
				MultiTenant: &mockMultiTenant{
					spaces: spaceNames,
				},
				SecretManagement: &mockSecMgr{},
			},
		}

		deployer, err = argocd.NewArgoCD(opts, &initInfo)
		Expect(err).Should(BeNil())

		err = nil
	})

	AfterEach(func() {
		err = deployer.DeleteProduct(ctx, productID)
		Expect(err).Should(BeNil())
		appProject := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: argoCDNamespace,
			},
		}
		err = testutils.WaitForDelete(k8sClient, appProject)
		Expect(err).Should(BeNil())
	})

	It("can create product", func() {
		err = deployer.CreateProduct(ctx, productID)
		Expect(err).Should(BeNil())

		appProject := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: argoCDNamespace,
			},
			Spec: argov1alpha1.AppProjectSpec{},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject)
		Expect(err).Should(BeNil())
		Expect(len(appProject.Spec.Destinations)).Should(Equal(6))
		Expect(len(appProject.Spec.ClusterResourceWhitelist)).Should(Equal(1))
		res := appProject.Spec.ClusterResourceWhitelist[0]
		Expect(res.Group).Should(Equal("*"))
		Expect(res.Kind).Should(Equal("*"))
	})

	It("can add product user", func() {
		err = deployer.CreateProduct(ctx, productID)
		Expect(err).Should(BeNil())

		err = deployer.AddProductUser(ctx, syncer.PermissionRequest{
			RequestScope: syncer.RequestScopeProduct,
			Resource: syncer.ResourceMetaData{
				Product: "",
				Name:    productID,
			},
			User:       userNames[0],
			Permission: syncer.Permission{},
		})
		Expect(err).Should(BeNil())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      argocd.ArgocdRBACConfigMapName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm)
		Expect(err).Should(BeNil())
		matchingStr := fmt.Sprintf("p, role:%s, projects, get, %s, allow", userNames[0], productID)
		Expect(strings.Contains(cm.Data[rbacKey], matchingStr)).Should(BeTrue())
	})

	It("can remove product user", func() {
		err = deployer.CreateProduct(ctx, productID)
		Expect(err).Should(BeNil())

		req := syncer.PermissionRequest{
			RequestScope: syncer.RequestScopeProduct,
			Resource: syncer.ResourceMetaData{
				Product: "",
				Name:    productID,
			},
			User:       userNames[0],
			Permission: syncer.Permission{},
		}

		err = deployer.AddProductUser(ctx, req)
		Expect(err).Should(BeNil())

		err = deployer.DeleteProductUser(ctx, req)
		Expect(err).Should(BeNil())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      argocd.ArgocdRBACConfigMapName,
				Namespace: argoCDNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(cm), cm)
		Expect(err).Should(BeNil())
		matchingStr := fmt.Sprintf("p, role:%s, projects, get, %s, allow", userNames[0], productID)
		Expect(strings.Contains(cm.Data[rbacKey], matchingStr)).ShouldNot(BeTrue())
	})

	It("can create app", func() {
		app := syncer.Application{
			ResourceMetaData: syncer.ResourceMetaData{
				Product: productID,
				Name:    appNames[0],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[0]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[0],
			},
			Destinations: []syncer.Space{
				{
					ResourceMetaData: syncer.ResourceMetaData{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: &syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		err = deployer.CreateApp(ctx, app)
		Expect(err).Should(BeNil())
	})

	It("can remove app", func() {
		app := syncer.Application{
			ResourceMetaData: syncer.ResourceMetaData{
				Product: productID,
				Name:    appNames[0],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[0]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[0],
			},
			Destinations: []syncer.Space{
				{
					ResourceMetaData: syncer.ResourceMetaData{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: &syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		err := deployer.CreateApp(ctx, app)
		Expect(err).Should(BeNil())

		err = deployer.DeleteApp(ctx, app)
		Expect(err).Should(BeNil())

		argoApp := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appNames[0],
				Namespace: argoCDNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoApp), argoApp)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())
	})

	It("will remove coderepo when coderepo is not used", func() {
		app := syncer.Application{
			ResourceMetaData: syncer.ResourceMetaData{
				Product: productID,
				Name:    appNames[0],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[0]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[0],
			},
			Destinations: []syncer.Space{
				{
					ResourceMetaData: syncer.ResourceMetaData{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: &syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		err := deployer.CreateApp(ctx, app)
		Expect(err).Should(BeNil())

		err = deployer.DeleteApp(ctx, app)
		Expect(err).Should(BeNil())

		err = deployer.CleanUp()
		Expect(err).Should(BeNil())

		codeRepo := &v1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      repoNames[0],
				Namespace: nautesNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(codeRepo), codeRepo)
		Expect(err).ShouldNot(BeNil())
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())
	})

	It("will update source in clean up", func() {
		err = deployer.CreateProduct(ctx, productID)
		Expect(err).Should(BeNil())

		app := syncer.Application{
			ResourceMetaData: syncer.ResourceMetaData{
				Product: productID,
				Name:    appNames[0],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[0]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[0],
			},
			Destinations: []syncer.Space{
				{
					ResourceMetaData: syncer.ResourceMetaData{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: &syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		err := deployer.CreateApp(ctx, app)
		Expect(err).Should(BeNil())

		err = deployer.CleanUp()
		Expect(err).Should(BeNil())

		appProject := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: argoCDNamespace,
			},
			Spec: argov1alpha1.AppProjectSpec{},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject)
		Expect(err).Should(BeNil())
		Expect(appProject.Spec.SourceRepos[0]).Should(Equal(app.Git.URL))
	})

})
