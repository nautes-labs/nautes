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

package datasource

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
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

type RuntimeDataSource struct {
	k8sClient           client.Client
	nautesNamespaceName string
	Clusters            map[string]nautescrd.Cluster
	CodeRepoProviders   map[string]nautescrd.CodeRepoProvider
	CodeRepos           map[string]nautescrd.CodeRepo
	DeploymentRuntimes  map[string]nautescrd.DeploymentRuntime
	Environments        map[string]nautescrd.Environment
	PipelineRuntimes    map[string]nautescrd.ProjectPipelineRuntime
	Product             *nautescrd.Product
	ProductCodeRepo     *nautescrd.CodeRepo
	PermissionMatrix    permissionMatrix
	Projects            map[string]nautescrd.Project
}

func (db *RuntimeDataSource) cacheNautesResoucesFromCluster(ctx context.Context) error {
	listOptInNautesNamespace := client.InNamespace(db.nautesNamespaceName)

	clusterList := &nautescrd.ClusterList{}
	codeRepoProviderList := &nautescrd.CodeRepoProviderList{}

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

func (db *RuntimeDataSource) cacheProductResourcesFromCluster(ctx context.Context, productName string) error {
	product := &nautescrd.Product{
		ObjectMeta: metav1.ObjectMeta{
			Name:      productName,
			Namespace: db.nautesNamespaceName,
		},
	}
	if err := db.k8sClient.Get(ctx, client.ObjectKeyFromObject(product), product); err != nil {
		return err
	}

	db.Product = product

	productCodeRepoList := nautescrd.CodeRepoList{}
	if err := db.k8sClient.List(ctx, &productCodeRepoList, client.MatchingLabels{nautescrd.LABEL_FROM_PRODUCT: product.Name}); err != nil {
		return err
	}

	if len(productCodeRepoList.Items) != 1 {
		return fmt.Errorf("found %d code repo by product name", len(productCodeRepoList.Items))
	}

	db.ProductCodeRepo = &productCodeRepoList.Items[0]

	listOptInProductNamespace := client.InNamespace(productName)

	envList := &nautescrd.EnvironmentList{}
	projectList := &nautescrd.ProjectList{}
	codeRepoList := &nautescrd.CodeRepoList{}
	deployRuntimeList := &nautescrd.DeploymentRuntimeList{}
	pipelineRuntimeList := &nautescrd.ProjectPipelineRuntimeList{}

	ObjLists := []client.ObjectList{envList, projectList, codeRepoList, deployRuntimeList, pipelineRuntimeList}
	for _, objList := range ObjLists {
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

func (db *RuntimeDataSource) cachePermissionMatrix(ctx context.Context) error {
	productName := db.Product.Name
	listOptInProductNamespace := client.InNamespace(productName)

	codeRepoBindingList := &nautescrd.CodeRepoBindingList{}
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

		projects := []string{}
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

func (db *RuntimeDataSource) convertCodeReposToRuntimeFormat() error {
	for name, codeRepo := range db.CodeRepos {
		URL, err := db.GetCodeRepoURL(name)
		if err != nil {
			return err
		}

		codeRepo.Spec.URL = URL

		codeRepo.ObjectMeta = metav1.ObjectMeta{
			Name: codeRepo.Name,
		}

		db.CodeRepos[name] = codeRepo
	}

	if db.ProductCodeRepo != nil {
		db.ProductCodeRepo.ObjectMeta = metav1.ObjectMeta{
			Name: db.ProductCodeRepo.Name,
		}
	}

	return nil
}

func (db *RuntimeDataSource) removeIllegalReposInRuntime() {
	db.removeIllegalReposInPipelineRuntime()
}

func (db *RuntimeDataSource) removeIllegalReposInPipelineRuntime() {
	for runtimeName, runtime := range db.PipelineRuntimes {
		project := runtime.Spec.Project
		repos := db.PermissionMatrix.ListCodeReposInProject(project)
		if len(repos) == 0 {
			delete(db.PipelineRuntimes, runtimeName)
		}

		repoSet := sets.New(repos...)
		if !repoSet.Has(runtime.Spec.PipelineSource) {
			delete(db.PipelineRuntimes, runtimeName)
		}

		newEventSources := []nautescrd.EventSource{}
		for _, envSrc := range runtime.Spec.EventSources {
			if envSrc.Gitlab == nil || repoSet.Has(envSrc.Gitlab.RepoName) {
				newEventSources = append(newEventSources, envSrc)
			}
		}

		runtime.Spec.EventSources = newEventSources
		removeEventSourceInPipelineRuntime(&runtime)

		db.PipelineRuntimes[runtimeName] = runtime
	}
}

func removeEventSourceInPipelineRuntime(runtime *nautescrd.ProjectPipelineRuntime) {
	eventSourceNames := map[string]bool{}
	for _, eventSrc := range runtime.Spec.EventSources {
		eventSourceNames[eventSrc.Name] = true
	}

	newTrigger := []nautescrd.PipelineTrigger{}
	for _, trigger := range runtime.Spec.PipelineTriggers {
		if !eventSourceNames[trigger.EventSource] {
			continue
		}
		newTrigger = append(newTrigger, trigger)
	}
	runtime.Spec.PipelineTriggers = newTrigger
}

func (db *RuntimeDataSource) GetProduct() (*nautescrd.Product, error) {
	if db.Product == nil {
		return nil, fmt.Errorf("product not found")
	}

	return db.Product.DeepCopy(), nil
}

func (db *RuntimeDataSource) GetCodeRepo(name string) (*nautescrd.CodeRepo, error) {
	if db.ProductCodeRepo != nil && db.ProductCodeRepo.Name == name {
		return db.ProductCodeRepo.DeepCopy(), nil
	}

	repo, ok := db.CodeRepos[name]
	if !ok {
		return nil, fmt.Errorf("get code repo %s failed", name)
	}
	return &repo, nil
}

func (db *RuntimeDataSource) GetCodeRepoURL(name string) (string, error) {
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

	url, err := url.JoinPath(provider.Spec.SSHAddress, db.Product.Spec.Name, codeRepo.Spec.RepoName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.git", url), nil
}

func (db *RuntimeDataSource) getListOptions(opts []ListOption) *ListOptions {
	options := &ListOptions{}
	for _, fn := range opts {
		fn(options)
	}
	return options
}

func (db *RuntimeDataSource) ListPipelineRuntimes() ([]nautescrd.ProjectPipelineRuntime, error) {
	runtimes := make([]nautescrd.ProjectPipelineRuntime, len(db.PipelineRuntimes))
	i := 0
	for _, runtime := range db.PipelineRuntimes {
		runtimes[i] = runtime
		i++
	}

	return runtimes, nil
}

func (db *RuntimeDataSource) ListRuntimes() []nautescrd.Runtime {
	runtimes := make([]nautescrd.Runtime, 0)

	for _, runtime := range db.DeploymentRuntimes {
		runtimes = append(runtimes, runtime.DeepCopy())
	}

	for _, runtime := range db.PipelineRuntimes {
		runtimes = append(runtimes, runtime.DeepCopy())
	}

	return runtimes
}

func (db *RuntimeDataSource) GetClusterByRuntime(runtime nautescrd.Runtime) (*nautescrd.Cluster, error) {
	env, ok := db.Environments[runtime.GetDestination()]
	if !ok {
		return nil, fmt.Errorf("get env %s failed", runtime.GetDestination())
	}

	cluster, ok := db.Clusters[env.Spec.Cluster]
	if !ok {
		return nil, fmt.Errorf("get cluster %s failed", env.Spec.Cluster)
	}

	return &cluster, nil
}

func (db *RuntimeDataSource) ListUsedNamespaces(opts ...ListOption) (NamespaceUsage, error) {
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

func (db *RuntimeDataSource) ListUsedCodeRepos(opts ...ListOption) ([]nautescrd.CodeRepo, error) {
	options := db.getListOptions(opts)

	repoSet := map[string]nautescrd.CodeRepo{}

	for _, runtime := range db.DeploymentRuntimes {
		cluster, err := db.GetClusterByRuntime(&runtime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(&runtime, cluster.Name) {
			continue
		}

		repo, err := db.GetCodeRepo(runtime.Spec.ManifestSource.CodeRepo)
		if err != nil {
			return nil, err
		}
		repoSet[repo.Name] = *repo
	}

	for _, runtime := range db.PipelineRuntimes {
		cluster, err := db.GetClusterByRuntime(&runtime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(&runtime, cluster.Name) {
			continue
		}

		if runtime.Spec.AdditionalResources != nil &&
			runtime.Spec.AdditionalResources.Git != nil &&
			runtime.Spec.AdditionalResources.Git.CodeRepo != "" {
			repo, err := db.GetCodeRepo(runtime.Spec.AdditionalResources.Git.CodeRepo)
			if err != nil {
				return nil, err
			}
			repoSet[repo.Name] = *repo
		}
	}

	repos := make([]nautescrd.CodeRepo, 0)
	for _, repo := range repoSet {
		repos = append(repos, repo)
	}

	if !options.withOutProductInfo {
		repos = append(repos, *db.ProductCodeRepo)
	}

	return repos, nil
}

func (db *RuntimeDataSource) ListUsedURLs(opts ...ListOption) ([]string, error) {
	options := db.getListOptions(opts)

	repos, err := db.ListUsedCodeRepos(opts...)
	if err != nil {
		return nil, err
	}

	URLs := []string{}
	for _, runtime := range db.PipelineRuntimes {
		cluster, err := db.GetClusterByRuntime(&runtime)
		if err != nil {
			return nil, err
		}

		if !options.MatchRequest(&runtime, cluster.Name) {
			continue
		}

		if runtime.Spec.AdditionalResources == nil ||
			runtime.Spec.AdditionalResources.Git == nil ||
			runtime.Spec.AdditionalResources.Git.URL == "" {
			continue
		}

		URLs = append(URLs, runtime.Spec.AdditionalResources.Git.URL)
	}

	for _, repo := range repos {
		URLs = append(URLs, repo.Spec.URL)
	}

	return URLs, nil
}

func NewRuntimeDataSource(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (DataSource, error) {
	return newRuntimeDataSource(ctx, k8sClient, productName, nautesNamespace)
}

func newRuntimeDataSource(ctx context.Context, k8sClient client.Client, productName string, nautesNamespace string) (*RuntimeDataSource, error) {
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

func getEmptyRuntimeDataSource() *RuntimeDataSource {
	return &RuntimeDataSource{
		Clusters:           map[string]nautescrd.Cluster{},
		CodeRepoProviders:  map[string]nautescrd.CodeRepoProvider{},
		CodeRepos:          map[string]nautescrd.CodeRepo{},
		DeploymentRuntimes: map[string]nautescrd.DeploymentRuntime{},
		Environments:       map[string]nautescrd.Environment{},
		PipelineRuntimes:   map[string]nautescrd.ProjectPipelineRuntime{},
		Projects:           map[string]nautescrd.Project{},
	}
}

func convertListToMap[T any](objs []T) map[string]T {
	objMap := make(map[string]T)

	for _, obj := range objs {
		rst := reflect.ValueOf(&obj).MethodByName("GetName").Call(nil)
		name := rst[0].String()
		objMap[name] = obj
	}

	return objMap
}
