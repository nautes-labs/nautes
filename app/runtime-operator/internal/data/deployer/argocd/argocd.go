package argocd

import (
	"context"
	"fmt"
	"reflect"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/deployer"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/initinfo"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/datasource"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const localCluster = "https://kubernetes.default.svc"

type ArgoCD struct {
	k8sClient       client.Client
	argoCDNamespace string
	productName     string
	clusterName     string
	resourceLabels  map[string]string
	db              datasource.DataSource
}

func NewDeployer(namespace string, initInfo initinfo.ComponentInitInfo, db datasource.DataSource) (deployer.Deployer, error) {
	if initInfo.ClusterConnectInfo.ClusterType != initinfo.ClusterTypeKubernetes {
		return nil, fmt.Errorf("argocd can only work on kubernetes")
	}

	cfg := initInfo.ClusterConnectInfo.Kubernetes.Config
	if initInfo.ClusterConnectInfo.Kubernetes.Config == nil {
		return nil, fmt.Errorf("rest config is null")
	}

	if err := corev1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}
	if err := argov1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	return &ArgoCD{
		k8sClient:       k8sClient,
		argoCDNamespace: namespace,
		productName:     initInfo.ProductName,
		clusterName:     initInfo.ClusterName,
		resourceLabels:  initInfo.Labels,
		db:              db,
	}, nil

}

func (d *ArgoCD) DeployApp(ctx context.Context, app deployer.Application) error {
	err := d.syncAppProject(ctx)
	if err != nil {
		return err
	}

	return d.deployApp(ctx, app)
}

func (d *ArgoCD) syncAppProject(ctx context.Context) error {
	project := d.getEmptyAppProject()

	sourceRepos, err := d.getSourceRepos()
	if err != nil {
		return err
	}

	destinations, err := d.getDestinations()
	if err != nil {
		return err
	}

	err = d.createOrUpdate(ctx, project, func() error {
		if err := utils.CheckResourceOperability(project, d.productName); err != nil {
			return err
		}

		project.Spec.SourceRepos = sourceRepos
		project.Spec.Destinations = destinations
		return nil
	})
	if err != nil {
		return fmt.Errorf("create app project %s failed: %w", project.Name, err)
	}

	return nil
}

func (d *ArgoCD) getSourceRepos() ([]string, error) {
	return d.db.ListUsedURLs(
		datasource.InCluster(d.clusterName),
		datasource.WithOutDeletedRuntimes(),
	)
}

func (d *ArgoCD) getDestinations() ([]argov1alpha1.ApplicationDestination, error) {
	namespaces, err := d.db.ListUsedNamespaces(
		datasource.InCluster(d.clusterName),
		datasource.WithOutDeletedRuntimes(),
	)
	if err != nil {
		return nil, fmt.Errorf("list namespaces in cluster %s failed: %w", d.clusterName, err)
	}

	destinations := []argov1alpha1.ApplicationDestination{}
	for _, namespace := range namespaces.GetNamespacesInCluster(d.clusterName) {
		destinations = append(destinations, argov1alpha1.ApplicationDestination{
			Server:    localCluster,
			Namespace: namespace,
		})
	}

	return destinations, nil
}

func (d *ArgoCD) deployApp(ctx context.Context, runtimeApp deployer.Application) error {
	if runtimeApp.Destination.Kubernetes == nil {
		return fmt.Errorf("get namespace in kubernetes failed")
	}

	app := d.getEmptyApp(runtimeApp.Name)

	err := d.createOrUpdate(ctx, app, func() error {
		if err := utils.CheckResourceOperability(app, d.productName); err != nil {
			return err
		}

		app.Spec.Source = argov1alpha1.ApplicationSource{
			RepoURL:        runtimeApp.Git.URL,
			Path:           runtimeApp.Git.Path,
			TargetRevision: runtimeApp.Git.Revision,
		}

		app.Spec.Destination = argov1alpha1.ApplicationDestination{
			Server:    localCluster,
			Namespace: runtimeApp.Destination.Kubernetes.Namespace,
			Name:      "",
		}

		app.Spec.Project = d.productName

		app.Spec.SyncPolicy = &argov1alpha1.SyncPolicy{
			Automated: &argov1alpha1.SyncPolicyAutomated{
				Prune:    true,
				SelfHeal: true,
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create app %s failed: %w", app.Name, err)
	}
	return nil
}

func (d *ArgoCD) UnDeployApp(ctx context.Context, app deployer.Application) error {
	err := d.deleteApp(ctx, app)
	if err != nil {
		return err
	}

	project := d.getEmptyAppProject()
	namespaces, err := d.db.ListUsedNamespaces(
		datasource.InCluster(d.clusterName),
		datasource.WithOutDeletedRuntimes(),
		datasource.WithOutProductInfo(),
	)
	if len(namespaces.GetNamespacesInCluster(d.clusterName)) == 0 {
		return client.IgnoreNotFound(d.k8sClient.Delete(ctx, project))
	}

	return d.syncAppProject(ctx)
}

func (d *ArgoCD) deleteApp(ctx context.Context, runtimeApp deployer.Application) error {
	app := d.getEmptyApp(runtimeApp.Name)
	if err := d.k8sClient.Get(ctx, client.ObjectKeyFromObject(app), app); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get app %s failed: %w", app.Name, err)
	}
	if err := utils.CheckResourceOperability(app, d.productName); err != nil {
		return err
	}

	return d.k8sClient.Delete(ctx, app)
}

func (d *ArgoCD) getEmptyAppProject() *argov1alpha1.AppProject {
	return &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.productName,
			Namespace: d.argoCDNamespace,
			Labels:    d.resourceLabels,
		},
		Spec: argov1alpha1.AppProjectSpec{},
	}
}

func (d *ArgoCD) getEmptyApp(name string) *argov1alpha1.Application {
	return &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.argoCDNamespace,
			Labels:    d.resourceLabels,
		},
		Spec: argov1alpha1.ApplicationSpec{},
	}
}

// argocd ApplicationDestination has unexport key, can not use controllerutil.CreateOrUpdate()
func (d *ArgoCD) createOrUpdate(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error {
	key := client.ObjectKeyFromObject(obj)
	if err := d.k8sClient.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := mutate(f, key, obj); err != nil {
			return err
		}
		return d.k8sClient.Create(ctx, obj)
	}

	existing := obj.DeepCopyObject()
	if err := mutate(f, key, obj); err != nil {
		return err
	}

	if reflect.DeepEqual(existing, obj) {
		return nil
	}

	return d.k8sClient.Update(ctx, obj)
}

// mutate wraps a MutateFn and applies validation to its result.
func mutate(f controllerutil.MutateFn, key client.ObjectKey, obj client.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey := client.ObjectKeyFromObject(obj); key != newKey {
		return fmt.Errorf("MutateFn cannot mutate object name and/or object namespace")
	}
	return nil
}
