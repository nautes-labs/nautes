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

package argocd

import (
	"context"
	"fmt"

	argocrd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	. "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	nautescontext "github.com/nautes-labs/nautes/pkg/context"

	argocdrbac "github.com/nautes-labs/nautes/app/runtime-operator/pkg/casbin/adapter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimecontext "github.com/nautes-labs/nautes/app/runtime-operator/pkg/context"
)

const (
	_ARGO_OPERATOR_ROLE_KEY     = "Argo"
	_ARGOCD_RBAC_ConfigMap_Name = "argocd-rbac-cm"
)

const (
	codeRepoUserDefault = "default"
)

type Syncer struct {
	K8sClient client.Client
}

var (
	kubernetesDefaultService = "https://kubernetes.default.svc"
)

func (d Syncer) Deploy(ctx context.Context, task interfaces.RuntimeSyncTask) (*interfaces.DeploymentDeploymentResult, error) {
	_, repoName, destCluster, appProject, app, appSource, policyManager, err := d.initNames(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("init var faled: %w", err)
	}

	repoURL, err := d.syncCodeRepo(ctx, task.Product.Name, repoName, *destCluster)
	if err != nil {
		return nil, fmt.Errorf("sync code repo failed: %w", err)
	}

	appSource.RepoURL = repoURL
	err = app.SyncApp(ctx, func(spec *argocrd.ApplicationSpec) {
		spec.Source = *appSource
		spec.Destination.Namespace = task.Product.Name
		spec.Project = appProject.GetName()
	})
	if err != nil {
		return nil, fmt.Errorf("sync argocd app %s failed: %w", app.GetName(), err)
	}

	deploymenRuntime := task.Runtime.(*nautescrd.DeploymentRuntime)
	spec, err := d.calcArgoCDAppProjectSpec(ctx, *deploymenRuntime, task.NautesCfg.Nautes.Namespace)
	if err != nil {
		return nil, err
	}
	if err := appProject.SyncAppProject(ctx, *spec); err != nil {
		return nil, fmt.Errorf("update argocd app project failed: %w", err)
	}

	if err := policyManager.AddRole(task.Product.Spec.Name, task.Product.Name, appProject.GetName()); err != nil {
		return nil, fmt.Errorf("add rbac policy failed: %w", err)
	}

	if err := policyManager.SavePolicyToCluster(types.NamespacedName{
		Namespace: appProject.GetNamespace(),
		Name:      _ARGOCD_RBAC_ConfigMap_Name,
	}); err != nil {
		return nil, fmt.Errorf("update rbac policy to cluster failed: %w", err)
	}

	return &interfaces.DeploymentDeploymentResult{Source: repoURL}, nil
}

func (d Syncer) UnDeploy(ctx context.Context, task interfaces.RuntimeSyncTask) error {
	_, repoName, destCluster, appProject, app, _, policyManager, err := d.initNames(ctx, task)
	if err != nil {
		return fmt.Errorf("init var faled: %w", err)
	}

	if err = app.Destroy(ctx); err != nil {
		return fmt.Errorf("delete argocd app failed: %w", err)
	}

	deletable, err := d.checkCodeRepoIsDeletable(ctx, task.Product.Name, repoName, task.Runtime.GetName())
	if err != nil {
		return fmt.Errorf("find delatable code repo failed: %w", err)
	}
	if deletable {
		if err = d.deleteCodeRepo(ctx, task.Product.Name, repoName, *destCluster); err != nil {
			return fmt.Errorf("remove code repo from dest cluster failed: %w", err)
		}
	}

	deploymenRuntime := task.Runtime.(*nautescrd.DeploymentRuntime)
	spec, err := d.calcArgoCDAppProjectSpec(ctx, *deploymenRuntime, task.NautesCfg.Nautes.Namespace)
	if err != nil {
		return err
	}
	if err := appProject.SyncAppProject(ctx, *spec); err != nil {
		return fmt.Errorf("update argocd app project failed: %w", err)
	}

	if appProject.isDeletable() {
		if err := appProject.destroy(ctx); err != nil {
			return fmt.Errorf("delete app project failed: %w", err)
		}
	}

	if err := policyManager.DeleteRole(task.Product.Name); err != nil {
		return fmt.Errorf("revoke project permission failed: %w", err)
	}

	if err := policyManager.SavePolicyToCluster(types.NamespacedName{
		Namespace: appProject.GetNamespace(),
		Name:      _ARGOCD_RBAC_ConfigMap_Name,
	}); err != nil {
		return fmt.Errorf("update rbac policy to cluster failed: %w", err)
	}

	return nil
}

func (d Syncer) initNames(ctx context.Context, task interfaces.RuntimeSyncTask) (
	history *nautescrd.DeployHistory,
	repoName string,
	cluster *destCluster,
	appProject *appProject,
	app *app,
	appSource *argocrd.ApplicationSource,
	rbacAdapter *argocdrbac.Adapter,
	err error) {
	deployRuntime, ok := task.Runtime.(*nautescrd.DeploymentRuntime)
	if !ok {
		err = fmt.Errorf("convert runtime to deployment runtime failed, unsupport runtime")
		return
	}
	history = deployRuntime.Status.DeployHistory
	repoName = deployRuntime.Spec.ManifestSource.CodeRepo

	k8sClient, err := client.New(task.AccessInfo.Kubernetes, client.Options{Scheme: scheme})
	if err != nil {
		return
	}
	cluster = &destCluster{
		name:        task.AccessInfo.Name,
		productName: task.Product.Name,
		Client:      k8sClient,
	}

	appName := task.Runtime.GetName()
	appSource = &argocrd.ApplicationSource{
		Path:           deployRuntime.Spec.ManifestSource.Path,
		TargetRevision: deployRuntime.Spec.ManifestSource.TargetRevision,
	}

	appProjectName := task.Runtime.GetProduct()
	argocdNamespace := task.NautesCfg.Deploy.ArgoCD.Namespace
	appProject, err = NewAppProject(appProjectName, argocdNamespace, task.Product.Name, k8sClient)
	if err != nil {
		err = fmt.Errorf("init argocd app project failed: %w", err)
		return
	}

	app, err = newApp(appName, argocdNamespace, task.Product.Name, k8sClient)
	if err != nil {
		err = fmt.Errorf("init argocd app failed: %w", err)
		return
	}

	rbacAdapter = argocdrbac.NewAdapter(
		func(ada *argocdrbac.Adapter) {
			ada.K8sClient = k8sClient
		})

	err = rbacAdapter.LoadPolicyFromCluster(types.NamespacedName{
		Namespace: argocdNamespace,
		Name:      _ARGOCD_RBAC_ConfigMap_Name,
	})

	return
}

func (d Syncer) syncCodeRepo(ctx context.Context, productName, repoName string, dest destCluster) (string, error) {
	cfg, err := nautescontext.FromConfigContext(ctx, ContextKeyConfig)
	if err != nil {
		return "", err
	}
	providerType := string(cfg.Git.GitType)

	codeRepo, err := d.getCodeRepo(ctx, productName, repoName)
	if err != nil {
		return "", fmt.Errorf("get coderepo failed: %w", err)
	}

	key := types.NamespacedName{
		Namespace: codeRepo.Namespace,
		Name:      codeRepo.Name,
	}

	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return "", fmt.Errorf("get secret client from context failed")
	}
	roleName, ok := cfg.Secret.OperatorName[_ARGO_OPERATOR_ROLE_KEY]
	if !ok {
		return "", fmt.Errorf("can not find argo operator name in nautes config")
	}

	secret := interfaces.SecretInfo{
		Type: interfaces.SECRET_TYPE_GIT,
		CodeRepo: &interfaces.CodeRepo{
			ProviderType: providerType,
			ID:           repoName,
			User:         codeRepoUserDefault,
			Permission:   interfaces.CodeRepoPermissionReadOnly,
		},
	}
	err = secClient.GrantPermission(ctx, secret, roleName, dest.name)
	if err != nil {
		return "", fmt.Errorf("grant permission to argo operator failed: %w", err)
	}

	destCodeRepo := &nautescrd.CodeRepo{}
	err = dest.GetClient().Get(ctx, key, destCodeRepo)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return "", err
		}

		// if not found in dest cluster create a copy from tenant cluster
		err = dest.GetClient().Create(ctx, codeRepo)
		if err != nil {
			return "", err
		}

		return codeRepo.Spec.URL, err
	}

	if reason, ok := isNotTerminatingAndBelongsToProduct(destCodeRepo, productName); !ok {
		return "", fmt.Errorf(reason)
	}

	if !isSameCodeRepo(codeRepo, destCodeRepo) {
		destCodeRepo.Spec = codeRepo.Spec
		return codeRepo.Spec.URL, dest.GetClient().Update(ctx, destCodeRepo)
	}

	return codeRepo.Spec.URL, nil
}

func (d Syncer) deleteCodeRepo(ctx context.Context, productName, repoName string, dest destCluster) error {
	cfg, err := nautescontext.FromConfigContext(ctx, ContextKeyConfig)
	if err != nil {
		return err
	}

	providerType := string(cfg.Git.GitType)
	codeRepo := &nautescrd.CodeRepo{}
	key := types.NamespacedName{
		Namespace: cfg.Nautes.Namespace,
		Name:      repoName,
	}
	if err = dest.GetClient().Get(ctx, key, codeRepo); err != nil {
		return client.IgnoreNotFound(err)
	}

	if name, ok := codeRepo.Labels[nautescrd.LABEL_BELONG_TO_PRODUCT]; !ok || name != productName {
		return fmt.Errorf("code repo not belongs to product %s", productName)
	}

	if err = dest.GetClient().Delete(ctx, codeRepo); err != nil {
		return err
	}

	secClient, ok := runtimecontext.FromSecretClientConetxt(ctx)
	if !ok {
		return fmt.Errorf("get secret client from context failed")
	}

	roleName, ok := cfg.Secret.OperatorName[_ARGO_OPERATOR_ROLE_KEY]
	if !ok {
		return fmt.Errorf("can not find argo operator name in nautes config")
	}

	secret := interfaces.SecretInfo{
		Type: interfaces.SECRET_TYPE_GIT,
		CodeRepo: &interfaces.CodeRepo{
			ProviderType: providerType,
			ID:           repoName,
			User:         codeRepoUserDefault,
			Permission:   interfaces.CodeRepoPermissionReadOnly,
		},
	}
	err = secClient.RevokePermission(ctx, secret, roleName, dest.name)
	if err != nil {
		return fmt.Errorf("revoke permission failed: %w", err)
	}

	return nil
}

func (d Syncer) checkCodeRepoIsDeletable(ctx context.Context, productName, repoName, runtimeName string) (bool, error) {
	runtimes := &nautescrd.DeploymentRuntimeList{}
	listOpts := []client.ListOption{
		client.InNamespace(productName),
	}
	err := d.K8sClient.List(ctx, runtimes, listOpts...)
	if err != nil {
		return false, err
	}

	isDeleteable := true
	for _, runtime := range runtimes.Items {
		if runtime.Spec.ManifestSource.CodeRepo == repoName &&
			runtime.Name != runtimeName {
			isDeleteable = false
			break
		}
	}

	return isDeleteable, nil
}

func (d Syncer) getCodeRepo(ctx context.Context, productName, name string) (*nautescrd.CodeRepo, error) {
	cfg, err := nautescontext.FromConfigContext(ctx, ContextKeyConfig)
	if err != nil {
		return nil, err
	}
	key := types.NamespacedName{
		Namespace: productName,
		Name:      name,
	}
	codeRepo := &nautescrd.CodeRepo{}

	if err := d.K8sClient.Get(ctx, key, codeRepo); err != nil {
		return nil, err
	}

	provider := &nautescrd.CodeRepoProvider{}
	providerKey := types.NamespacedName{
		Namespace: cfg.Nautes.Namespace,
		Name:      codeRepo.Spec.CodeRepoProvider,
	}
	err = d.K8sClient.Get(ctx, providerKey, provider)
	if err != nil {
		return nil, fmt.Errorf("get code repo provider failed: %w", err)
	}

	product := &nautescrd.Product{}
	productKey := types.NamespacedName{
		Namespace: cfg.Nautes.Namespace,
		Name:      productName,
	}
	err = d.K8sClient.Get(ctx, productKey, product)
	if err != nil {
		return nil, fmt.Errorf("get product failed: %w", err)
	}

	newRepo := &nautescrd.CodeRepo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      codeRepo.Name,
			Namespace: cfg.Nautes.Namespace,
			Labels:    map[string]string{nautescrd.LABEL_BELONG_TO_PRODUCT: productName},
		},
		Spec: codeRepo.Spec,
	}
	newRepo.Spec.URL = fmt.Sprintf("%s/%s/%s.git", provider.Spec.SSHAddress, product.Spec.Name, codeRepo.Spec.RepoName)
	return newRepo, nil
}

func (d Syncer) calcArgoCDAppProjectSpec(ctx context.Context, runtime nautescrd.DeploymentRuntime, nautesNamespace string) (*argocrd.AppProjectSpec, error) {
	productID := runtime.GetProduct()
	cluster, err := utils.GetClusterByRuntime(ctx, d.K8sClient, nautesNamespace, &runtime)
	if err != nil {
		return nil, fmt.Errorf("get runtime %s's cluster failed: %w", runtime.GetName(), err)
	}

	reservedNamespaces := utils.GetReservedNamespacesByProduct(*cluster, productID)
	productNamespaces, err := utils.GetProductNamespacesInCluster(ctx, d.K8sClient, productID, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("get product namespace in cluster %s failed: %w", cluster.Name, err)
	}
	appProjectNamespaces := append(reservedNamespaces, productNamespaces...)

	urls, err := utils.GetURLsInCluster(ctx, d.K8sClient, productID, cluster.Name, nautesNamespace)
	if err != nil {
		return nil, fmt.Errorf("get product URLs in cluster %s failed: %w", cluster.Name, err)
	}

	appProjectSpec := &argocrd.AppProjectSpec{
		SourceRepos:              urls,
		Destinations:             getAppProjectDestinations(appProjectNamespaces),
		ClusterResourceWhitelist: getAppProjectClusterResourceWhitelist(*cluster, productID),
	}

	return appProjectSpec, nil
}

func getAppProjectDestinations(destinations []string) []argocrd.ApplicationDestination {
	appProjectDestinations := []argocrd.ApplicationDestination{}
	for _, namespace := range destinations {
		appProjectDestinations = append(appProjectDestinations, argocrd.ApplicationDestination{
			Server:    "*",
			Namespace: namespace,
		})
	}

	return appProjectDestinations
}

func getAppProjectClusterResourceWhitelist(cluster nautescrd.Cluster, productID string) []metav1.GroupKind {
	var clusterResourcesWhiteList []metav1.GroupKind
	if cluster.Spec.ProductAllowedClusterResources != nil {
		productName := utils.GetProductNameByIDInCluster(cluster, productID)
		clusterResources, ok := cluster.Spec.ProductAllowedClusterResources[productName]
		if ok {
			for _, clusterResource := range clusterResources {
				clusterResourcesWhiteList = append(clusterResourcesWhiteList, metav1.GroupKind{
					Group: clusterResource.Group,
					Kind:  clusterResource.Kind,
				})
			}
		}
	}
	return clusterResourcesWhiteList
}

func isSameCodeRepo(from, to *nautescrd.CodeRepo) bool {
	if from.Spec.URL != to.Spec.URL {
		return false
	}
	return true
}

func isNotTerminatingAndBelongsToProduct(res client.Object, productName string) (string, bool) {
	if !res.GetDeletionTimestamp().IsZero() {
		return fmt.Sprintf("resouce %s is terminating", res.GetName()), false
	}
	labels := res.GetLabels()
	name, ok := labels[nautescrd.LABEL_BELONG_TO_PRODUCT]
	if !ok || name != productName {
		return fmt.Sprintf("resource %s is not belongs to product", res.GetName()), false
	}
	return "", true
}
