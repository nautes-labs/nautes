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

package database

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// permissionMatrix record permission matrix in product, format map[project]repos
type permissionMatrix map[string][]string

func (b permissionMatrix) AppendCodeRepo(projectName, repoName string) {
	repos, ok := b[projectName]
	if !ok {
		return
	}

	repos = append(repos, repoName)
	b[projectName] = repos
}

func (b permissionMatrix) AppendProject(name string) {
	if _, ok := b[name]; ok {
		return
	}
	b[name] = []string{}
}

func (b permissionMatrix) ListProjects() []string {
	projects := make([]string, 0, len(b))
	for project := range b {
		projects = append(projects, project)
	}
	return projects
}

func (b permissionMatrix) ListCodeReposInProject(name string) []string {
	return b[name]
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projects,verbs=get;list;watch

type RuntimeDataBase struct {
	k8sClient           client.Client
	nautesNamespaceName string
	Clusters            map[string]v1alpha1.Cluster
	CodeRepoProviders   map[string]v1alpha1.CodeRepoProvider
	CodeRepos           map[string]v1alpha1.CodeRepo
	DeploymentRuntimes  map[string]v1alpha1.DeploymentRuntime
	Environments        map[string]v1alpha1.Environment
	PipelineRuntimes    map[string]v1alpha1.ProjectPipelineRuntime
	Product             *v1alpha1.Product
	ProductCodeRepo     *v1alpha1.CodeRepo
	PermissionMatrix    permissionMatrix
	Projects            map[string]v1alpha1.Project
}

func (db *RuntimeDataBase) cacheNautesResoucesFromCluster(ctx context.Context) error {
	listOptInNautesNamespace := client.InNamespace(db.nautesNamespaceName)

	clusterList := &v1alpha1.ClusterList{}
	codeRepoProviderList := &v1alpha1.CodeRepoProviderList{}

	nautesObjLists := []client.ObjectList{clusterList, codeRepoProviderList}
	for _, objList := range nautesObjLists {
		if err := db.k8sClient.List(ctx, objList, listOptInNautesNamespace); err != nil {
			return err
		}
	}

	db.Clusters = convertListToMap(clusterList.Items)
	db.CodeRepoProviders = convertListToMap(codeRepoProviderList.Items)

	return nil
}

func (db *RuntimeDataBase) cacheProductResourcesFromCluster(ctx context.Context, productName string) error {
	product := &v1alpha1.Product{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: db.nautesNamespaceName,
		},
	}
	if err := db.k8sClient.Get(ctx, client.ObjectKeyFromObject(product), product); err != nil {
		return err
	}

	db.Product = product

	productCodeRepoList := v1alpha1.CodeRepoList{}
	if err := db.k8sClient.List(ctx, &productCodeRepoList, client.MatchingLabels{v1alpha1.LABEL_FROM_PRODUCT: product.Name}); err != nil {
		return err
	}

	if len(productCodeRepoList.Items) != 1 {
		return fmt.Errorf("found %d code repo by product name", len(productCodeRepoList.Items))
	}

	db.ProductCodeRepo = &productCodeRepoList.Items[0]

	listOptInProductNamespace := client.InNamespace(productName)

	envList := &v1alpha1.EnvironmentList{}
	projectList := &v1alpha1.ProjectList{}
	codeRepoList := &v1alpha1.CodeRepoList{}
	deployRuntimeList := &v1alpha1.DeploymentRuntimeList{}
	pipelineRuntimeList := &v1alpha1.ProjectPipelineRuntimeList{}

	objLists := []client.ObjectList{envList, projectList, codeRepoList, deployRuntimeList, pipelineRuntimeList}
	for _, objList := range objLists {
		if err := db.k8sClient.List(ctx, objList, listOptInProductNamespace); err != nil {
			return err
		}
	}

	db.Environments = convertListToMap(envList.Items)
	db.Projects = convertListToMap(projectList.Items)
	db.CodeRepos = convertListToMap(codeRepoList.Items)
	db.DeploymentRuntimes = convertListToMap(deployRuntimeList.Items)
	db.PipelineRuntimes = convertListToMap(pipelineRuntimeList.Items)

	return nil
}

func (db *RuntimeDataBase) cachePermissionMatrix(ctx context.Context) error {
	productName := db.Product.Name
	listOptInProductNamespace := client.InNamespace(productName)

	codeRepoBindingList := &v1alpha1.CodeRepoBindingList{}
	if err := db.k8sClient.List(ctx, codeRepoBindingList, listOptInProductNamespace); err != nil {
		return err
	}

	matrix := permissionMatrix{}
	for _, project := range db.Projects {
		matrix.AppendProject(project.Name)
	}

	for _, rb := range codeRepoBindingList.Items {
		if rb.Spec.Product != productName {
			continue
		}

		var projects []string
		if len(rb.Spec.Projects) == 0 {
			projects = matrix.ListProjects()
		} else {
			projects = rb.Spec.Projects
		}

		for _, project := range projects {
			matrix.AppendCodeRepo(project, rb.Spec.CodeRepo)
		}
	}

	for _, repo := range db.CodeRepos {
		if repo.Spec.Project == "" {
			continue
		}
		matrix.AppendCodeRepo(repo.Spec.Project, repo.Name)
	}

	db.PermissionMatrix = matrix

	return nil
}

func (db *RuntimeDataBase) convertCodeReposToRuntimeFormat() error {
	for name, codeRepo := range db.CodeRepos {
		codeRepoURL, err := db.GetCodeRepoURL(name)
		if err != nil {
			return err
		}

		codeRepo.Spec.URL = codeRepoURL

		codeRepo.ObjectMeta = metav1.ObjectMeta{
			Name: codeRepo.Name,
		}

		db.CodeRepos[name] = codeRepo
	}

	if db.ProductCodeRepo != nil {
		db.ProductCodeRepo.ObjectMeta = metav1.ObjectMeta{
			Name:   db.ProductCodeRepo.Name,
			Labels: db.ProductCodeRepo.Labels,
		}
	}

	return nil
}

func (db *RuntimeDataBase) removeIllegalReposInRuntime() {
	db.removeIllegalReposInPipelineRuntime()
}

func (db *RuntimeDataBase) removeIllegalReposInPipelineRuntime() {
	for runtimeName, runtime := range db.PipelineRuntimes {
		newRuntime := runtime.DeepCopy()
		project := newRuntime.Spec.Project
		repos := db.PermissionMatrix.ListCodeReposInProject(project)
		if len(repos) == 0 {
			delete(db.PipelineRuntimes, runtimeName)
		}

		repoSet := sets.New(repos...)
		if !repoSet.Has(newRuntime.Spec.PipelineSource) {
			delete(db.PipelineRuntimes, runtimeName)
		}

		newEventSources := []v1alpha1.EventSource{}
		for _, envSrc := range newRuntime.Spec.EventSources {
			if envSrc.Gitlab == nil || repoSet.Has(envSrc.Gitlab.RepoName) {
				newEventSources = append(newEventSources, envSrc)
			}
		}

		newRuntime.Spec.EventSources = newEventSources
		removeEventSourceInPipelineRuntime(newRuntime)

		db.PipelineRuntimes[runtimeName] = *newRuntime
	}
}

func removeEventSourceInPipelineRuntime(runtime *v1alpha1.ProjectPipelineRuntime) {
	eventSourceNames := map[string]bool{}
	for _, eventSrc := range runtime.Spec.EventSources {
		eventSourceNames[eventSrc.Name] = true
	}

	newTrigger := []v1alpha1.PipelineTrigger{}
	for _, trigger := range runtime.Spec.PipelineTriggers {
		if !eventSourceNames[trigger.EventSource] {
			continue
		}
		newTrigger = append(newTrigger, trigger)
	}
	runtime.Spec.PipelineTriggers = newTrigger
}

func (db *RuntimeDataBase) GetProduct(name string) (*v1alpha1.Product, error) {
	if db.Product == nil || db.Product.Name != name {
		return nil, fmt.Errorf("product not found")
	}

	return db.Product.DeepCopy(), nil
}

func (db *RuntimeDataBase) GetProductCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	if db.ProductCodeRepo.Labels[v1alpha1.LABEL_FROM_PRODUCT] != name {
		return nil, fmt.Errorf("product coderepo not found")
	}

	return db.ProductCodeRepo.DeepCopy(), nil
}

func (db *RuntimeDataBase) GetCodeRepoProvider(name string) (*v1alpha1.CodeRepoProvider, error) {
	provider, ok := db.CodeRepoProviders[name]
	if !ok {
		return nil, fmt.Errorf("code repo provider %s not found", name)
	}
	return provider.DeepCopy(), nil
}

func (db *RuntimeDataBase) GetCodeRepo(name string) (*v1alpha1.CodeRepo, error) {
	if db.ProductCodeRepo != nil && db.ProductCodeRepo.Name == name {
		return db.ProductCodeRepo.DeepCopy(), nil
	}

	repo, ok := db.CodeRepos[name]
	if !ok {
		return nil, fmt.Errorf("get code repo %s failed", name)
	}
	return repo.DeepCopy(), nil
}

func (db *RuntimeDataBase) GetCodeRepoByURL(codeRepoURL string) (*v1alpha1.CodeRepo, error) {
	for _, coderepo := range db.CodeRepos {
		if coderepo.Spec.URL == codeRepoURL {
			return coderepo.DeepCopy(), nil
		}
	}
	return nil, fmt.Errorf("coderepo not found")
}

func (db *RuntimeDataBase) GetCodeRepoURL(name string) (string, error) {
	codeRepo, err := db.GetCodeRepo(name)
	if err != nil {
		return "", err
	}

	if codeRepo.Spec.URL != "" {
		return codeRepo.Spec.URL, nil
	}

	provider, ok := db.CodeRepoProviders[codeRepo.Spec.CodeRepoProvider]
	if !ok {
		return "", fmt.Errorf("code repo provider %s not found", codeRepo.Spec.CodeRepoProvider)
	}

	codeRepoURL, err := url.JoinPath(provider.Spec.SSHAddress, db.Product.Spec.Name, codeRepo.Spec.RepoName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.git", codeRepoURL), nil
}

func (db *RuntimeDataBase) getListOptions(opts []ListOption) *ListOptions {
	options := &ListOptions{}
	for _, fn := range opts {
		fn(options)
	}
	return options
}

func (db *RuntimeDataBase) ListPipelineRuntimes() ([]v1alpha1.ProjectPipelineRuntime, error) {
	runtimes := make([]v1alpha1.ProjectPipelineRuntime, len(db.PipelineRuntimes))
	i := 0
	for _, runtime := range db.PipelineRuntimes {
		runtimes[i] = runtime
		i++
	}

	return runtimes, nil
}

func (db *RuntimeDataBase) GetRuntime(name string, runtimeType v1alpha1.RuntimeType) (v1alpha1.Runtime, error) {
	switch runtimeType {
	case v1alpha1.RuntimeTypeDeploymentRuntime:
		runtime, ok := db.DeploymentRuntimes[name]
		if !ok {
			return nil, fmt.Errorf("runtime %s not found", name)
		}
		return runtime.DeepCopy(), nil
	case v1alpha1.RuntimeTypePipelineRuntime:
		runtime, ok := db.PipelineRuntimes[name]
		if !ok {
			return nil, fmt.Errorf("runtime %s not found", name)
		}
		return runtime.DeepCopy(), nil
	default:
		return nil, fmt.Errorf("unknow runtime type %s", runtimeType)
	}
}

func (db *RuntimeDataBase) ListRuntimes() []v1alpha1.Runtime {
	runtimes := make([]v1alpha1.Runtime, 0)

	for _, runtime := range db.DeploymentRuntimes {
		runtimes = append(runtimes, runtime.DeepCopy())
	}

	for _, runtime := range db.PipelineRuntimes {
		runtimes = append(runtimes, runtime.DeepCopy())
	}

	return runtimes
}

func (db *RuntimeDataBase) GetCluster(name string) (*v1alpha1.Cluster, error) {
	cluster, ok := db.Clusters[name]
	if !ok {
		return nil, fmt.Errorf("get cluster %s failed", name)
	}
	return cluster.DeepCopy(), nil
}

func (db *RuntimeDataBase) GetClusterByRuntime(runtime v1alpha1.Runtime) (*v1alpha1.Cluster, error) {
	env, ok := db.Environments[runtime.GetDestination()]
	if !ok {
		return nil, fmt.Errorf("get env %s failed", runtime.GetDestination())
	}

	return db.GetCluster(env.Spec.Cluster)
}

func (db *RuntimeDataBase) ListUsedNamespaces(opts ...ListOption) (NamespaceUsage, error) {
	options := db.getListOptions(opts)

	runtimes := db.ListRuntimes()

	namespaceUsage := NamespaceUsage{}
	for _, runtime := range runtimes {
		cluster, err := db.GetClusterByRuntime(runtime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(runtime, cluster.Name) {
			continue
		}

		namespaceUsage[cluster.Name] = append(namespaceUsage[cluster.Name], runtime.GetNamespaces()...)
	}

	if !options.withOutProductInfo {
		for cluster, namespaces := range namespaceUsage {
			namespaceUsage[cluster] = append(namespaces, db.Product.Name)
		}
	}

	return namespaceUsage, nil
}

func (db *RuntimeDataBase) ListUsedCodeRepos(opts ...ListOption) ([]v1alpha1.CodeRepo, error) {
	options := db.getListOptions(opts)

	repoSet := map[string]v1alpha1.CodeRepo{}

	deploymentUsedCodeRepos, err := db.listDeploymentRuntimeUsedCodeRepos(*options)
	if err != nil {
		return nil, err
	}

	for i, repo := range deploymentUsedCodeRepos {
		repoSet[repo.Name] = deploymentUsedCodeRepos[i]
	}

	pipelineUsedCodeRepos, err := db.listPipelineRuntimeUsedCodeRepos(*options)
	if err != nil {
		return nil, err
	}

	for i, repo := range pipelineUsedCodeRepos {
		repoSet[repo.Name] = pipelineUsedCodeRepos[i]
	}

	repos := make([]v1alpha1.CodeRepo, 0)
	for _, repo := range repoSet {
		repos = append(repos, repo)
	}

	if !options.withOutProductInfo {
		repos = append(repos, *db.ProductCodeRepo)
	}

	return repos, nil
}

func (db *RuntimeDataBase) listDeploymentRuntimeUsedCodeRepos(options ListOptions) ([]v1alpha1.CodeRepo, error) {
	repoNameSet := sets.New[string]()
	var repos []v1alpha1.CodeRepo
	for _, runtime := range db.DeploymentRuntimes {
		deployRuntime := runtime.DeepCopy()
		repoName := deployRuntime.Spec.ManifestSource.CodeRepo
		if repoNameSet.Has(repoName) {
			continue
		}

		cluster, err := db.GetClusterByRuntime(deployRuntime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(deployRuntime, cluster.Name) {
			continue
		}

		repo, err := db.GetCodeRepo(repoName)
		if err != nil {
			return nil, err
		}

		repoNameSet.Insert(repoName)
		repos = append(repos, *repo)
	}
	return repos, nil
}

func (db *RuntimeDataBase) listPipelineRuntimeUsedCodeRepos(options ListOptions) ([]v1alpha1.CodeRepo, error) {
	repoNameSet := sets.New[string]()
	var repos []v1alpha1.CodeRepo

	for _, runtime := range db.PipelineRuntimes {
		pipelineRuntime := runtime.DeepCopy()
		repoName := ""

		if pipelineRuntime.Spec.AdditionalResources != nil &&
			pipelineRuntime.Spec.AdditionalResources.Git != nil &&
			pipelineRuntime.Spec.AdditionalResources.Git.CodeRepo != "" {
			repoName = pipelineRuntime.Spec.AdditionalResources.Git.CodeRepo
		}
		if repoName == "" || repoNameSet.Has(repoName) {
			continue
		}

		cluster, err := db.GetClusterByRuntime(pipelineRuntime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(pipelineRuntime, cluster.Name) {
			continue
		}

		repo, err := db.GetCodeRepo(pipelineRuntime.Spec.AdditionalResources.Git.CodeRepo)
		if err != nil {
			return nil, err
		}

		repoNameSet.Insert(repoName)
		repos = append(repos, *repo)
	}

	return repos, nil
}

func NewRuntimeDataSource(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (Database, error) {
	return newRuntimeDataSource(ctx, k8sClient, productName, nautesNamespace)
}

func newRuntimeDataSource(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (*RuntimeDataBase, error) {
	dataSource := getEmptyRuntimeDataSource()
	dataSource.k8sClient = k8sClient
	dataSource.nautesNamespaceName = nautesNamespace

	if err := dataSource.cacheNautesResoucesFromCluster(ctx); err != nil {
		return nil, err
	}

	if err := dataSource.cacheProductResourcesFromCluster(ctx, productName); err != nil {
		return nil, err
	}

	if err := dataSource.convertCodeReposToRuntimeFormat(); err != nil {
		return nil, err
	}

	if err := dataSource.cachePermissionMatrix(ctx); err != nil {
		return nil, err
	}

	dataSource.removeIllegalReposInRuntime()

	return dataSource, nil
}

func getEmptyRuntimeDataSource() *RuntimeDataBase {
	return &RuntimeDataBase{
		Clusters:           map[string]v1alpha1.Cluster{},
		CodeRepoProviders:  map[string]v1alpha1.CodeRepoProvider{},
		CodeRepos:          map[string]v1alpha1.CodeRepo{},
		DeploymentRuntimes: map[string]v1alpha1.DeploymentRuntime{},
		Environments:       map[string]v1alpha1.Environment{},
		PipelineRuntimes:   map[string]v1alpha1.ProjectPipelineRuntime{},
		Projects:           map[string]v1alpha1.Project{},
	}
}

func convertListToMap[T any](objs []T) map[string]T {
	objMap := make(map[string]T)

	for i := range objs {
		rst := reflect.ValueOf(&objs[i]).MethodByName("GetName").Call(nil)
		name := rst[0].String()
		objMap[name] = objs[i]
	}

	return objMap
}
