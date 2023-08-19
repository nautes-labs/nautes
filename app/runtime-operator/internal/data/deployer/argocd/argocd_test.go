package argocd_test

import (
	"context"
	"fmt"
	"reflect"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployer/argocd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/deployer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/initinfo"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/datasource"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const LabelBelongToProduct = "resource.nautes.io/belongsto"

var _ = Describe("", func() {
	var task initinfo.ComponentInitInfo
	var namespaces datasource.NamespaceUsage
	var urls []string
	var argoCDNamespace *corev1.Namespace
	var argoDeployer deployer.Deployer
	var seed string
	var namespaceNames []string
	var ctx context.Context
	BeforeEach(func() {
		seed = RandNum()
		ctx = context.Background()

		task = initinfo.ComponentInitInfo{
			ClusterConnectInfo: initinfo.ClusterConnectInfo{ClusterType: initinfo.ClusterTypeKubernetes, Kubernetes: &initinfo.ClusterConnectInfoKubernetes{Config: cfg}},
			ProductName:        fmt.Sprintf("product-%s", seed),
			ClusterName:        fmt.Sprintf("cluster-%s", seed),
		}
		task.Labels = map[string]string{
			LabelBelongToProduct: task.ProductName,
		}

		argoCDNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("argons-%s", seed),
			},
		}
		err := k8sClient.Create(ctx, argoCDNamespace)
		Expect(err).Should(BeNil())

		namespaceNames = GenerateNames(fmt.Sprintf("ns-%%d-%s", seed), 5)
		clusterName := task.ClusterName
		namespaces = datasource.NamespaceUsage{
			clusterName: namespaceNames,
		}

		urls = GenerateNames(fmt.Sprintf("ssh://127.0.0.1/%s/repo-%%d.git", seed), 10)

		mockDB := &mockDB{
			Namespaces: &namespaces,
			URLs:       urls,
		}

		argoDeployer, err = argocd.NewDeployer(argoCDNamespace.Name, task, mockDB)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		err := k8sClient.Delete(ctx, argoCDNamespace)
		Expect(err).Should(BeNil())
	})

	It("can create app project and app by task", func() {
		appName := fmt.Sprintf("app-%s", seed)
		deployApp := deployer.Application{
			Name: appName,
			Git: &deployer.ApplicationGit{
				URL:      urls[1],
				Revision: "main",
				Path:     "/pipeline",
			},
			Destination: deployer.Destination{
				Kubernetes: &deployer.DestinationKubernetes{
					Namespace: namespaceNames[1],
				},
			},
		}
		err := argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		// Test deploying app repeated
		err = argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		project := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.ProductName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.Destinations)).Should(Equal(len(namespaceNames)))
		Expect(project.ObjectMeta.Labels[LabelBelongToProduct]).Should(Equal(task.ProductName))
		Expect(reflect.DeepEqual(project.Spec.SourceRepos, urls)).Should(BeTrue())

		app := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		Expect(app.ObjectMeta.Labels[LabelBelongToProduct]).Should(Equal(task.ProductName))
		Expect(app.Spec.Destination.Namespace).Should(Equal(namespaceNames[1]))
		Expect(app.Spec.Source.RepoURL).Should(Equal(urls[1]))
		Expect(app.Spec.Source.Path).Should(Equal("/pipeline"))
		Expect(app.Spec.Source.TargetRevision).Should(Equal("main"))
	})

	It("task is change, it can update app", func() {
		appName := fmt.Sprintf("app-%s", seed)
		deployApp := deployer.Application{
			Name: appName,
			Git: &deployer.ApplicationGit{
				URL:      urls[1],
				Revision: "main",
				Path:     "/pipeline",
			},
			Destination: deployer.Destination{
				Kubernetes: &deployer.DestinationKubernetes{
					Namespace: namespaceNames[1],
				},
			},
		}
		err := argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		deployApp.Git.URL = urls[2]
		err = argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		app := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		Expect(app.Spec.Source.RepoURL).Should(Equal(urls[2]))

		deployApp.Destination.Kubernetes.Namespace = namespaceNames[2]
		err = argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).Should(BeNil())
		Expect(app.Spec.Destination.Namespace).Should(Equal(namespaceNames[2]))
	})

	It("will delete app project when namespaces is nil", func() {
		appName := fmt.Sprintf("app-%s", seed)
		deployApp := deployer.Application{
			Name: appName,
			Git: &deployer.ApplicationGit{
				URL:      urls[1],
				Revision: "main",
				Path:     "/pipeline",
			},
			Destination: deployer.Destination{
				Kubernetes: &deployer.DestinationKubernetes{
					Namespace: namespaceNames[1],
				},
			},
		}
		err := argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		namespaces = nil

		err = argoDeployer.UnDeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		project := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.ProductName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).ShouldNot(BeNil())
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

		app := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).ShouldNot(BeNil())
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

	})

	It("will not delete app project when namespaces is not nil", func() {
		appName := fmt.Sprintf("app-%s", seed)
		deployApp := deployer.Application{
			Name: appName,
			Git: &deployer.ApplicationGit{
				URL:      urls[1],
				Revision: "main",
				Path:     "/pipeline",
			},
			Destination: deployer.Destination{
				Kubernetes: &deployer.DestinationKubernetes{
					Namespace: namespaceNames[1],
				},
			},
		}
		err := argoDeployer.DeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		namespaces[task.ClusterName] = append(namespaces[task.ClusterName][:3], namespaces[task.ClusterName][4:]...)

		err = argoDeployer.UnDeployApp(ctx, deployApp)
		Expect(err).Should(BeNil())

		project := &argov1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      task.ProductName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(project), project)
		Expect(err).Should(BeNil())
		Expect(len(project.Spec.Destinations)).Should(Equal(len(namespaces[task.ClusterName])))

		app := &argov1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      appName,
				Namespace: argoCDNamespace.Name,
			},
		}

		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app)
		Expect(err).ShouldNot(BeNil())
		Expect(apierrors.IsNotFound(err)).Should(BeTrue())

	})
})
