package argocd

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	argocdrbac "github.com/nautes-labs/nautes/app/runtime-operator/pkg/casbin/adapter"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	argov1alpha1 "github.com/nautes-labs/nautes/pkg/thirdpartapis/argocd/application/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func init() {
	utilruntime.Must(argov1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

type argocd struct {
	db                       database.Database
	k8sClient                client.Client
	components               *syncer.ComponentList
	namespace                string
	nautesNamespace          string
	opts                     map[string]string
	secretUser               syncer.User
	cluster                  v1alpha1.Cluster
	policyManager            argocdrbac.Adapter
	projectsNeedUpdateSource sets.Set[string]
}

func NewArgoCD(opt v1alpha1.Component, info *syncer.ComponentInitInfo) (syncer.Deployment, error) {
	if info.ClusterConnectInfo.Type != v1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.Type)
	}

	k8sClient, err := client.New(info.ClusterConnectInfo.Kubernetes.Config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	cluster, err := info.NautesDB.GetCluster(info.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("get cluster failed: %w", err)
	}

	secUser := syncer.User{
		Resource: syncer.Resource{Name: info.NautesConfig.Secret.OperatorName[configs.OperatorNameArgo]},
		UserType: syncer.UserTypeMachine,
		AuthInfo: &syncer.Auth{
			Kubernetes: []syncer.AuthKubernetes{
				{
					ServiceAccount: info.NautesConfig.Nautes.ServiceAccount[configs.OperatorNameArgo],
					Namespace:      info.NautesConfig.Nautes.Namespace,
				},
			},
		},
	}

	rbacAdapter := argocdrbac.NewAdapter(
		func(ada *argocdrbac.Adapter) {
			ada.K8sClient = k8sClient
		})

	argoCD := &argocd{
		db:                       info.NautesDB,
		k8sClient:                k8sClient,
		components:               info.Components,
		namespace:                opt.Namespace,
		nautesNamespace:          info.NautesConfig.Nautes.Namespace,
		opts:                     opt.Additions,
		secretUser:               secUser,
		cluster:                  *cluster,
		policyManager:            *rbacAdapter,
		projectsNeedUpdateSource: sets.Set[string]{},
	}

	return argoCD, nil
}

func (a *argocd) CleanUp() error {
	ctx := context.TODO()

	for _, project := range a.projectsNeedUpdateSource.UnsortedList() {
		if err := a.updateAppProjectSources(ctx, project); err != nil {
			return fmt.Errorf("update app project source failed")
		}
	}

	codeRepoList := &v1alpha1.CodeRepoList{}
	if err := a.k8sClient.List(ctx, codeRepoList, client.InNamespace(a.nautesNamespace)); err != nil {
		return fmt.Errorf("list code repos failed: %w", err)
	}

	for _, codeRepo := range codeRepoList.Items {
		if _, ok := codeRepo.Labels[v1alpha1.LABEL_TENANT_MANAGEMENT]; ok {
			continue
		}

		userList := newUserList(codeRepo.Annotations[AnnotationCodeRepoUsers])
		if len(userList.getUsers()) == 0 {
			if err := a.deleteCodeRepo(ctx, &codeRepo); err != nil {
				return fmt.Errorf("delete code repo %s failed: %w", codeRepo.Name, err)
			}
		}
	}
	return nil
}

const (
	KubernetesAPIServerAddr = "https://kubernetes.default.svc"
)

func (a *argocd) CreateProduct(ctx context.Context, name string) error {
	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(name),
		},
	}

	spaces, err := a.components.MultiTenant.ListSpaces(ctx, name, syncer.IgnoreResourceInDeletion())
	if err != nil {
		return fmt.Errorf("list spaces failed: %w", err)
	}

	_, err = controllerutil.CreateOrUpdate(ctx, a.k8sClient, appProject, func() error {
		if err := utils.CheckResourceOperability(appProject, name); err != nil {
			return err
		}

		appProject.Spec.Destinations = make([]argov1alpha1.ApplicationDestination, 0)

		for _, space := range spaces {
			appProject.Spec.Destinations = append(appProject.Spec.Destinations, argov1alpha1.ApplicationDestination{
				Server:    KubernetesAPIServerAddr,
				Namespace: space.Kubernetes.Namespace,
			})
		}

		for productName, id := range a.cluster.Status.ProductIDMap {
			if id != name {
				continue
			}

			appProject.Spec.ClusterResourceWhitelist = a.getClusterResourceWhiteList(productName)
			for _, ns := range a.getReservedNamespaces(productName) {
				appProject.Spec.Destinations = append(appProject.Spec.Destinations, argov1alpha1.ApplicationDestination{
					Server:    KubernetesAPIServerAddr,
					Namespace: ns,
				})
			}
		}

		return nil
	})

	return err
}

func (a *argocd) DeleteProduct(ctx context.Context, name string) error {
	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(name),
		},
	}
	if err := a.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		return client.IgnoreNotFound(err)
	}
	if !utils.IsBelongsToProduct(appProject, name) {
		return fmt.Errorf("app project %s is not belongs to product %s", name, name)
	}

	return a.k8sClient.Delete(ctx, appProject)
}

func (a *argocd) GetProduct(_ context.Context, _ string) (*syncer.ProductStatus, error) {
	panic("not implemented") // TODO: Implement
}

const (
	ArgocdRBACConfigMapName = "argocd-rbac-cm"
)

func (a *argocd) AddProductUser(_ context.Context, request syncer.PermissionRequest) error {
	switch request.RequestScope {
	case syncer.RequestScopeProduct:
		if err := a.policyManager.LoadPolicyFromCluster(types.NamespacedName{
			Namespace: a.namespace,
			Name:      ArgocdRBACConfigMapName,
		}); err != nil {
			return fmt.Errorf("load policy from cluster failed: %w", err)
		}

		if err := a.policyManager.AddRole(request.User, request.User, request.Resource.Name); err != nil {
			return fmt.Errorf("add rbac policy failed: %w", err)
		}

		if err := a.policyManager.SavePolicyToCluster(types.NamespacedName{
			Namespace: a.namespace,
			Name:      ArgocdRBACConfigMapName,
		}); err != nil {
			return fmt.Errorf("update rbac policy to cluster failed: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("scope type %s is not supported", request.RequestScope)
	}
}

func (a *argocd) DeleteProductUser(_ context.Context, request syncer.PermissionRequest) error {
	switch request.RequestScope {
	case syncer.RequestScopeProduct:
		if err := a.policyManager.LoadPolicyFromCluster(types.NamespacedName{
			Namespace: a.namespace,
			Name:      ArgocdRBACConfigMapName,
		}); err != nil {
			return fmt.Errorf("load policy from cluster failed: %w", err)
		}

		if err := a.policyManager.DeleteRole(request.User); err != nil {
			return fmt.Errorf("add rbac policy failed: %w", err)
		}

		if err := a.policyManager.SavePolicyToCluster(types.NamespacedName{
			Namespace: a.namespace,
			Name:      ArgocdRBACConfigMapName,
		}); err != nil {
			return fmt.Errorf("update rbac policy to cluster failed: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("scope type %s is not supported", request.RequestScope)
	}
}

func (a *argocd) CreateApp(ctx context.Context, app syncer.Application) error {
	if err := a.createOrUpdateArgoCDApp(ctx, app); err != nil {
		return err
	}

	a.projectsNeedUpdateSource.Insert(app.Product)

	if app.Git != nil && app.Git.CodeRepo != "" {
		return a.createOrUpdateCodeRepo(ctx, app.Git.CodeRepo, []string{a.secretUser.Name}, nil)
	}

	return nil
}

func (a *argocd) DeleteApp(ctx context.Context, app syncer.Application) error {
	if app.Git != nil && app.Git.CodeRepo != "" {
		if err := a.createOrUpdateCodeRepo(ctx, app.Git.CodeRepo, nil, []string{a.secretUser.Name}); err != nil {
			return err
		}
	}

	if err := a.deleteArgoCDApp(ctx, app.Product, app.Name); err != nil {
		return err
	}

	a.projectsNeedUpdateSource.Insert(app.Product)

	return nil
}

func (a *argocd) getClusterResourceWhiteList(name string) []metav1.GroupKind {
	whiteListFromCluster, ok := a.cluster.Spec.ProductAllowedClusterResources[name]
	if !ok {
		return nil
	}

	whiteList := make([]metav1.GroupKind, len(whiteListFromCluster))
	for i := 0; i < len(whiteListFromCluster); i++ {
		whiteList[i] = metav1.GroupKind{
			Group: whiteListFromCluster[i].Group,
			Kind:  whiteListFromCluster[i].Kind,
		}
	}
	return whiteList
}

func (a *argocd) getReservedNamespaces(name string) []string {
	var namespaces []string
	for namespace, allowedProducts := range a.cluster.Spec.ReservedNamespacesAllowedProducts {
		productSet := sets.New(allowedProducts...)
		if productSet.Has(name) {
			namespaces = append(namespaces, namespace)
		}
	}
	return namespaces
}

func (a *argocd) createOrUpdateArgoCDApp(ctx context.Context, app syncer.Application) error {
	if len(app.Destinations) == 0 {
		return fmt.Errorf("destinations is empty")
	}

	argoApp := &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(app.Product),
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, a.k8sClient, argoApp, func() error {
		if err := utils.CheckResourceOperability(argoApp, app.Product); err != nil {
			return err
		}

		argoApp.Spec.Source = argov1alpha1.ApplicationSource{
			RepoURL:        app.Git.URL,
			Path:           app.Git.Path,
			TargetRevision: app.Git.Revision,
		}

		argoApp.Spec.Destination = argov1alpha1.ApplicationDestination{
			Server:    KubernetesAPIServerAddr,
			Namespace: app.Destinations[0].Kubernetes.Namespace,
		}

		argoApp.Spec.Project = app.Product

		argoApp.Spec.SyncPolicy = &argov1alpha1.SyncPolicy{
			Automated: &argov1alpha1.SyncPolicyAutomated{
				Prune:    true,
				SelfHeal: true,
			},
		}

		return nil
	})
	return err
}

func (a *argocd) deleteArgoCDApp(ctx context.Context, productName, name string) error {
	argoApp := &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
		},
	}

	return a.deleteObject(ctx, productName, argoApp)
}

func (a *argocd) deleteObject(ctx context.Context, productName string, obj client.Object) error {
	if err := a.k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	if !utils.IsBelongsToProduct(obj, productName) {
		return fmt.Errorf("app %s is not belongs to product %s", obj.GetName(), productName)
	}

	return a.k8sClient.Delete(ctx, obj)
}

const (
	AnnotationCodeRepoUsers = "CodeRepoUsers"
)

func (a *argocd) createOrUpdateCodeRepo(ctx context.Context, name string, newUsers, deleteUsers []string) error {
	repo, err := a.db.GetCodeRepo(name)
	if err != nil {
		return err
	}

	provider, err := a.db.GetCodeRepoProvider(repo.Spec.CodeRepoProvider)
	if err != nil {
		return err
	}

	if len(newUsers) != 0 {
		repoPermissionReq := syncer.SecretInfo{
			Type: syncer.SecretTypeCodeRepo,
			CodeRepo: &syncer.CodeRepo{
				ProviderType: provider.Spec.ProviderType,
				ID:           repo.Name,
				User:         "default",
				Permission:   syncer.CodeRepoPermissionReadOnly,
			},
		}
		if err := a.components.SecretManagement.GrantPermission(ctx, repoPermissionReq, a.secretUser); err != nil {
			return fmt.Errorf("grant code repo %s permission to argo operator failed: %w", repo.Name, err)
		}
	}

	repo.Namespace = a.nautesNamespace
	_, err = controllerutil.CreateOrUpdate(ctx, a.k8sClient, repo, func() error {
		if repo.Annotations == nil {
			repo.Annotations = map[string]string{}
		}

		userList := newUserList(repo.Annotations[AnnotationCodeRepoUsers])
		userList.addUsers(newUsers)
		userList.deleteUsers(deleteUsers)

		repo.Annotations[AnnotationCodeRepoUsers] = userList.getUsersAsString()

		return nil
	})
	return err
}

func (a *argocd) deleteCodeRepo(ctx context.Context, codeRepo *v1alpha1.CodeRepo) error {
	provider, err := a.db.GetCodeRepoProvider(codeRepo.Spec.CodeRepoProvider)
	if err != nil {
		return err
	}

	repoPermissionReq := syncer.SecretInfo{
		Type: syncer.SecretTypeCodeRepo,
		CodeRepo: &syncer.CodeRepo{
			ProviderType: provider.Spec.ProviderType,
			ID:           codeRepo.Name,
			User:         "default",
			Permission:   syncer.CodeRepoPermissionReadOnly,
		},
	}

	if err := a.components.SecretManagement.RevokePermission(ctx, repoPermissionReq, a.secretUser); err != nil {
		return fmt.Errorf("revoke code repo %s from argo operator failed: %w", codeRepo.Name, err)
	}

	return a.k8sClient.Delete(ctx, codeRepo)
}

// updateAppProjectSources list apps in project and update project's SourceRepos
func (a *argocd) updateAppProjectSources(ctx context.Context, name string) error {
	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(name),
		},
	}

	if err := a.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		return client.IgnoreNotFound(err)
	}

	if err := utils.CheckResourceOperability(appProject, name); err != nil {
		return err
	}

	appList := &argov1alpha1.ApplicationList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(utils.GetProductLabel(name)),
		client.InNamespace(a.namespace),
	}
	if err := a.k8sClient.List(ctx, appList, listOpts...); err != nil {
		return err
	}

	var repoURLs []string
	for _, app := range appList.Items {
		if app.Spec.Source.RepoURL != "" {
			repoURLs = append(repoURLs, app.Spec.Source.RepoURL)
		}
	}

	appProject.Spec.SourceRepos = repoURLs
	return a.k8sClient.Update(ctx, appProject)
}
