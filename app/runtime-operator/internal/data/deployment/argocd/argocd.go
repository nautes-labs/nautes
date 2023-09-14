package argocd

import (
	"context"
	"fmt"
	"reflect"

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2"
	argocdrbac "github.com/nautes-labs/nautes/app/runtime-operator/pkg/casbin/adapter"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	db              database.Database
	k8sClient       client.Client
	multiTenant     syncer.MultiTenant
	namespace       string
	nautesNamespace string
	opts            map[string]string
	secretMgr       syncer.SecretManagement
	secretUser      syncer.User
	cluster         v1alpha1.Cluster
	policyManager   argocdrbac.Adapter
}

func NewArgoCD(opt v1alpha1.Component, info syncer.ComponentInitInfo) (syncer.Deployment, error) {
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
		db:              info.NautesDB,
		k8sClient:       k8sClient,
		multiTenant:     info.Components.MultiTenant,
		namespace:       opt.Namespace,
		nautesNamespace: info.NautesConfig.Nautes.Namespace,
		opts:            opt.Additions,
		secretMgr:       info.Components.SecretManagement,
		secretUser:      secUser,
		cluster:         *cluster,
		policyManager:   *rbacAdapter,
	}

	return argoCD, nil
}

// When the component generates cache information, implement this method to clean datas.
// This method will be automatically called by the syncer after each tuning is completed.
func (a *argocd) CleanUp() error {
	return nil
}

const (
	KubernetesApiServerAddr = "https://kubernetes.default.svc"
)

func (a *argocd) CreateProduct(ctx context.Context, name string, _ interface{}) (interface{}, error) {
	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(name),
		},
	}

	spaces, err := a.multiTenant.ListSpaces(ctx, name, syncer.IgnoreResourceInDeletion())
	if err != nil {
		return nil, fmt.Errorf("list spaces failed: %w", err)
	}

	_, err = controllerutil.CreateOrUpdate(ctx, a.k8sClient, appProject, func() error {
		if err := utils.CheckResourceOperability(appProject, name); err != nil {
			return err
		}

		appProject.Spec.Destinations = make([]argov1alpha1.ApplicationDestination, 0)

		for _, space := range spaces {
			appProject.Spec.Destinations = append(appProject.Spec.Destinations, argov1alpha1.ApplicationDestination{
				Server:    KubernetesApiServerAddr,
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
					Server:    KubernetesApiServerAddr,
					Namespace: ns,
				})
			}
		}

		return nil
	})

	return nil, err
}

func (a *argocd) DeleteProduct(ctx context.Context, name string, _ interface{}) (interface{}, error) {
	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(name),
		},
	}
	if err := a.k8sClient.Get(ctx, client.ObjectKeyFromObject(appProject), appProject); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	if !utils.IsBelongsToProduct(appProject, name) {
		return nil, fmt.Errorf("app project %s is not belongs to product %s", name, name)
	}

	return nil, a.k8sClient.Delete(ctx, appProject)
}

func (a *argocd) GetProduct(ctx context.Context, name string) (*syncer.ProductStatus, error) {
	panic("not implemented") // TODO: Implement
}

const (
	ArgocdRBACConfigMapName = "argocd-rbac-cm"
)

func (a *argocd) AddProductUser(ctx context.Context, request syncer.PermissionRequest) error {
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

func (a *argocd) DeleteUProductUser(ctx context.Context, request syncer.PermissionRequest) error {
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

// SyncApp should deploy the given apps, and clean up expired apps in cache.
// All apps share one cache.
func (a *argocd) SyncApp(ctx context.Context, apps []syncer.Application, cache interface{}) (interface{}, error) {
	oldCache, err := newAppCache(cache)
	if err != nil {
		return cache, err
	}
	newCache, _ := newAppCache(cache)

	deleteApps := getDeleteApps(apps, *newCache)
	if err := a.syncApps(ctx, applications(apps), deleteApps, newCache); err != nil {
		return newCache, fmt.Errorf("sync apps failed: %w", err)
	}

	newCodeReposUsage := applications(apps).GetCodeRepoUsage()
	oldCodeReposUsage := newCache.CodeReposUsage
	codeRepoUserChangedInfos := getCodeRepoUserChangedInfos(newCodeReposUsage, oldCodeReposUsage)

	if err := a.syncCodeRepos(ctx, codeRepoUserChangedInfos, newCache); err != nil {
		return newCache, fmt.Errorf("sync code repos failed: %w", err)
	}

	projectsNeedUpdate := getProjectsFromApps(applications(apps).GetApps(), oldCache.AppNames)
	for _, project := range projectsNeedUpdate {
		if err := a.updateAppProjectSources(ctx, project); err != nil {
			return newCache, fmt.Errorf("update project %s sources failed: %w", project, err)
		}
	}

	return newCache, nil
}

func (a *argocd) SyncAppUsers(_ context.Context, _ []syncer.PermissionRequest, cache interface{}) (interface{}, error) {
	return cache, nil
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

func (a *argocd) syncApps(ctx context.Context, apps applications, deleteApps []string, cache *AppCache) error {
	for _, app := range apps {
		if err := a.createOrUpdateApp(ctx, app); err != nil {
			return fmt.Errorf("create or udate app failed: %w", err)
		}

		index := appIndex{
			Product: app.Product,
			Name:    app.Name,
		}
		cache.AddApp(index.GetIndex())
	}

	for _, app := range deleteApps {
		appIndex := newAppIndex(app)
		if err := a.deleteApp(ctx, appIndex.Product, appIndex.Name); err != nil {
			return err
		}
		cache.DeleteApp(app)
	}

	return nil
}

func (a *argocd) createOrUpdateApp(ctx context.Context, app syncer.Application) error {
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

	return a.createOrUpdate(ctx, argoApp, func() error {
		if err := utils.CheckResourceOperability(argoApp, app.Product); err != nil {
			return err
		}

		argoApp.Spec.Source = argov1alpha1.ApplicationSource{
			RepoURL:        app.Git.URL,
			Path:           app.Git.Path,
			TargetRevision: app.Git.Revision,
		}

		argoApp.Spec.Destination = argov1alpha1.ApplicationDestination{
			Server:    KubernetesApiServerAddr,
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
}

func (a *argocd) deleteApp(ctx context.Context, productName, name string) error {
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

type codeRepoUserChangedInfo struct {
	new    []string
	delete []string
}

func getCodeRepoUserChangedInfos(newUsage, oldUsage CodeReposUsage) map[string]codeRepoUserChangedInfo {
	newCodeRepoSet := newUsage.GetCodeRepoSet()
	oldCodeRepoSet := oldUsage.GetCodeRepoSet()

	allRepos := newCodeRepoSet.Union(oldCodeRepoSet)

	infos := map[string]codeRepoUserChangedInfo{}
	for _, repo := range allRepos.UnsortedList() {
		newUserSet := sets.New(newUsage[repo].getUsers()...)
		oldUserSet := sets.New(oldUsage[repo].getUsers()...)

		newUsers := newUserSet.Difference(oldUserSet).UnsortedList()
		deleteUsers := oldUserSet.Difference(newUserSet).UnsortedList()

		infos[repo] = codeRepoUserChangedInfo{
			new:    newUsers,
			delete: deleteUsers,
		}
	}
	return infos
}

func (a *argocd) syncCodeRepos(ctx context.Context, userChangedInfos map[string]codeRepoUserChangedInfo, cache *AppCache) error {
	for codeRepo, changedInfo := range userChangedInfos {
		cache.CodeReposUsage.AddCodeRepoUsage(codeRepo, changedInfo.new)
		cache.CodeReposUsage.DeleteCodeRepoUsage(codeRepo, changedInfo.delete)
		if err := a.createOrUpdateCodeRepo(ctx, codeRepo, changedInfo.new, changedInfo.delete); err != nil {
			return fmt.Errorf("update code repo %s's users failed: %w", codeRepo, err)
		}
	}

	codeRepoList := &v1alpha1.CodeRepoList{}
	if err := a.k8sClient.List(ctx, codeRepoList, client.InNamespace(a.nautesNamespace)); err != nil {
		return fmt.Errorf("list code repos failed: %w", err)
	}

	for _, codeRepo := range codeRepoList.Items {
		userList := newUserList(codeRepo.Annotations[AnnotationCodeRepoUsers])
		if len(userList.getUsers()) == 0 {
			if err := a.deleteCodeRepo(ctx, &codeRepo); err != nil {
				return fmt.Errorf("delete code repo %s failed: %w", codeRepo.Name, err)
			}
		}
	}

	return nil
}

func (a *argocd) createOrUpdateCodeRepo(ctx context.Context, name string, newUsers, deleteUsers []string) error {
	repo, err := a.db.GetCodeRepo(name)
	if err != nil {
		return err
	}

	provider, err := a.db.GetCodeRepoProvider(repo.Spec.CodeRepoProvider)
	if err != nil {
		return err
	}

	repoPermissionReq := syncer.SecretInfo{
		Type: syncer.SecretTypeCodeRepo,
		CodeRepo: &syncer.CodeRepo{
			ProviderType: provider.Spec.ProviderType,
			ID:           repo.Name,
			User:         "default",
			Permission:   syncer.CodeRepoPermissionReadOnly,
		},
	}
	if err := a.secretMgr.GrantPermission(ctx, repoPermissionReq, a.secretUser); err != nil {
		return fmt.Errorf("grant code repo %s permission to argo operator failed: %w", repo.Name, err)
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

	if err := a.secretMgr.RevokePermission(ctx, repoPermissionReq, a.secretUser); err != nil {
		return fmt.Errorf("revoke code repo %s from argo operator failed: %w", codeRepo.Name, err)
	}

	return a.k8sClient.Delete(ctx, codeRepo)
}

func getProjectsFromApps(newApps, oldApps []string) []string {
	newAppSet := sets.New(newApps...)
	oldAppSet := sets.New(oldApps...)

	projectSet := sets.New[string]()
	for _, index := range newAppSet.Union(oldAppSet).UnsortedList() {
		appIndex := newAppIndex(index)
		projectSet.Insert(appIndex.Product)
	}

	return projectSet.UnsortedList()
}

func (a *argocd) updateAppProjectSources(ctx context.Context, productName string) error {
	appList := &argov1alpha1.ApplicationList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(utils.GetProductLabel(productName)),
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

	appProject := &argov1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: a.namespace,
			Labels:    utils.GetProductLabel(productName),
		},
	}

	return a.createOrUpdate(ctx, appProject, func() error {
		if err := utils.CheckResourceOperability(appProject, productName); err != nil {
			return err
		}

		appProject.Spec.SourceRepos = repoURLs
		return nil
	})
}

// argocd ApplicationDestination has unexport key, can not use controllerutil.CreateOrUpdate()
func (a *argocd) createOrUpdate(ctx context.Context, obj client.Object, f controllerutil.MutateFn) error {
	key := client.ObjectKeyFromObject(obj)
	if err := a.k8sClient.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := mutate(f, key, obj); err != nil {
			return err
		}
		return a.k8sClient.Create(ctx, obj)
	}

	existing := obj.DeepCopyObject()
	if err := mutate(f, key, obj); err != nil {
		return err
	}

	if reflect.DeepEqual(existing, obj) {
		return nil
	}

	return a.k8sClient.Update(ctx, obj)
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

func convertAppsToSet(apps []syncer.Application) sets.Set[string] {
	appSet := sets.New[string]()
	for _, app := range apps {
		index := appIndex(app.Resource)
		appSet.Insert(index.GetIndex())
	}
	return appSet
}

func getDeleteApps(apps []syncer.Application, cache AppCache) []string {
	newAppSet := convertAppsToSet(apps)
	oldAppSet := sets.New(cache.AppNames...)
	return oldAppSet.Difference(newAppSet).UnsortedList()
}
