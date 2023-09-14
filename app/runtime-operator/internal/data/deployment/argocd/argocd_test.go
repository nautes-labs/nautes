package argocd_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployment/argocd"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
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
				Type: v1alpha1.CLUSTER_KIND_KUBERNETES,
				Kubernetes: &syncer.ClusterConnectInfoKubernetes{
					Config: restCFG,
				},
			},
			ClusterName: clusterName,
			RuntimeType: "",
			NautesDB:    db,
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
			Components: syncer.ComponentList{
				MultiTenant: &mockMultiTenant{
					spaces: spaceNames,
				},
				SecretManagement: &mockSecMgr{},
			},
		}

		deployer, err = argocd.NewArgoCD(opts, initInfo)
		Expect(err).Should(BeNil())

		err = nil
	})

	AfterEach(func() {
		_, err = deployer.DeleteProduct(ctx, productID, nil)
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
		_, err = deployer.CreateProduct(ctx, productID, nil)
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
		_, err = deployer.CreateProduct(ctx, productID, nil)
		Expect(err).Should(BeNil())

		err = deployer.AddProductUser(ctx, syncer.PermissionRequest{
			RequestScope: syncer.RequestScopeProduct,
			Resource: syncer.Resource{
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

	It("can create app", func() {
		app := syncer.Application{
			Resource: syncer.Resource{
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
					Resource: syncer.Resource{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		cache, err := deployer.SyncApp(ctx, []syncer.Application{app}, nil)
		Expect(err).Should(BeNil())

		appCache := cache.(*argocd.AppCache)
		destApp := fmt.Sprintf("%s/%s", productID, appNames[0])
		Expect(appCache.AppNames[0]).Should(Equal(destApp))
		destUsage := argocd.CodeReposUsage{
			repoNames[0]: {
				Name:  repoNames[0],
				Users: []string{appNames[0]},
			},
		}
		Expect(reflect.DeepEqual(appCache.CodeReposUsage, destUsage)).Should(BeTrue())

		argoApp := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appNames[0],
				Namespace: argoCDNamespace,
			},
		}

		destAppSpec := argov1alpha1.ApplicationSpec{
			Source: argov1alpha1.ApplicationSource{
				RepoURL:        app.Git.URL,
				Path:           app.Git.Path,
				TargetRevision: app.Git.Revision,
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    argocd.KubernetesApiServerAddr,
				Namespace: spaceNames[0],
			},
			Project: productID,
			SyncPolicy: &argov1alpha1.SyncPolicy{
				Automated: &argov1alpha1.SyncPolicyAutomated{
					Prune:      true,
					SelfHeal:   true,
					AllowEmpty: false,
				},
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoApp), argoApp)
		Expect(err).Should(BeNil())
		Expect(reflect.DeepEqual(argoApp.Spec, destAppSpec)).Should(BeTrue())

		appProject := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: argoCDNamespace,
			},
			Spec: argov1alpha1.AppProjectSpec{},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject)
		Expect(err).Should(BeNil())
		Expect(len(appProject.Spec.SourceRepos)).Should(Equal(1))
		Expect(appProject.Spec.SourceRepos[0]).Should(Equal(app.Git.URL))
	})

	It("can create many app once", func() {
		app := syncer.Application{
			Resource: syncer.Resource{
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
					Resource: syncer.Resource{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}
		app2 := syncer.Application{
			Resource: syncer.Resource{
				Product: productID,
				Name:    appNames[1],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[1]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[1],
			},
			Destinations: []syncer.Space{
				{
					Resource: syncer.Resource{
						Product: productID,
						Name:    spaceNames[1],
					},
					SpaceType: "",
					Kubernetes: syncer.SpaceKubernetes{
						Namespace: spaceNames[1],
					},
				},
			},
		}

		cache, err := deployer.SyncApp(ctx, []syncer.Application{app, app2}, nil)
		Expect(err).Should(BeNil())

		appCache := cache.(*argocd.AppCache)
		Expect(len(appCache.AppNames)).Should(Equal(2))
		destUsage := argocd.CodeRepoUsage{
			Name:  repoNames[0],
			Users: []string{appNames[0]},
		}
		Expect(reflect.DeepEqual(destUsage, appCache.CodeReposUsage[repoNames[0]])).Should(BeTrue())
		destUsage = argocd.CodeRepoUsage{
			Name:  repoNames[1],
			Users: []string{appNames[1]},
		}
		Expect(reflect.DeepEqual(destUsage, appCache.CodeReposUsage[repoNames[1]])).Should(BeTrue())

		app3 := syncer.Application{
			Resource: syncer.Resource{
				Product: productID,
				Name:    appNames[2],
			},
			Git: &syncer.ApplicationGit{
				URL:      db.CodeRepos[repoNames[0]].Spec.URL,
				Revision: "main",
				Path:     "./dest",
				CodeRepo: repoNames[0],
			},
			Destinations: []syncer.Space{
				{
					Resource: syncer.Resource{
						Product: productID,
						Name:    spaceNames[2],
					},
					SpaceType: "",
					Kubernetes: syncer.SpaceKubernetes{
						Namespace: spaceNames[2],
					},
				},
			},
		}

		cache, err = deployer.SyncApp(ctx, []syncer.Application{app, app3}, cache)
		Expect(err).Should(BeNil())
		appCache = cache.(*argocd.AppCache)
		Expect(len(appCache.AppNames)).Should(Equal(2))
		Expect(len(appCache.CodeReposUsage[repoNames[0]].Users)).Should(Equal(2))

	})

	It("can remove app", func() {
		app := syncer.Application{
			Resource: syncer.Resource{
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
					Resource: syncer.Resource{
						Product: productID,
						Name:    spaceNames[0],
					},
					SpaceType: "",
					Kubernetes: syncer.SpaceKubernetes{
						Namespace: spaceNames[0],
					},
				},
			},
		}

		cache, err := deployer.SyncApp(ctx, []syncer.Application{app}, nil)
		Expect(err).Should(BeNil())

		cache, err = deployer.SyncApp(ctx, nil, cache)
		Expect(err).Should(BeNil())

		appCache := cache.(*argocd.AppCache)
		Expect(len(appCache.AppNames)).Should(Equal(0))
		Expect(len(appCache.CodeReposUsage)).Should(Equal(0))

		argoApp := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appNames[0],
				Namespace: argoCDNamespace,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(argoApp), argoApp)
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		appProject := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      productID,
				Namespace: argoCDNamespace,
			},
			Spec: argov1alpha1.AppProjectSpec{},
		}
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject)
		Expect(err).Should(BeNil())
		Expect(len(appProject.Spec.SourceRepos)).Should(Equal(0))

	})

})
