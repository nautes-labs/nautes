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
	"fmt"

	errors "github.com/go-kratos/kratos/v2/errors"
	enviromentv1 "github.com/nautes-labs/nautes/api/api-server/environment/v1"
	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Argo-operator Index key, Used to obtain roles from the configuration
	ArgoOperator = "Argo"
	// Product default project name
	DefaultProject        = "default.project"
	KustomizationFileName = "kustomization.yaml"
)

const (
	// When pushing code configuration, there is a key with a context retry.
	RretryCount RretryCountType = "RretryCount"
)

const (
	// Product naming prefix
	ProductPrefix = "product-"
	// CodeRepo naming prefix
	RepoPrefix = "repo-"
)

// Secret store Constants.
const (
	// Secret Repo stores git related data engine name
	SecretsGitEngine = "git"
	// Secret Repo stores deploykey key
	SecretsDeployKey = "deploykey"
	// Secret Repo stores access token key
	SecretsAccessToken = "accesstoken"
	// Key data default store user
	DefaultUser = "default"
	// DeployKey fingerprint data Key
	Fingerprint = "fingerprint"
	// DeployKey fingerprint data ID
	DeployKeyID = "id"
)

const (
	// Git read-only and read-write permissions
	ReadOnly  DeployKeyType = "readonly"
	ReadWrite DeployKeyType = "readwrite"
)

const (
	// AccessToken fingerprint data ID
	AccessTokenID = "id"
	// Access token key path name
	AccessTokenName = "accesstoken-api"
	// Access token scope permissions
	APIPermission AccessTokenPermission = "api"
	// Access token authorization role
	Developer  AccessLevelValue = 30
	Maintainer AccessLevelValue = 40
	Owner      AccessLevelValue = 50
)

const (
	// layout resource directory names.
	CodeReposSubDir = "code-repos"
	RuntimesDir     = "runtimes"
	EnvSubDir       = "envs"
	ProjectsDir     = "projects"
)

type RescourceContext string

// Constants used to store commit information.
const (
	ResourceInfoKey RescourceContext = "ResourceInfoKey"
	// mark this request method
	SaveMethod   = "Save"
	DeleteMethod = "Delete"
)

// This is initial page options.
const (
	PerPage = 40
	Page    = 1
)

type ResourceInfo struct {
	// The product name to which the resource belongs
	ProductName string
	// Method of operating resources
	Method string
	// Parent class resource kind
	ParentResouceKind string
	// Parent class resource name
	ParentResourceName string
	// Current resource kind
	ResourceKind string
	// Current resource name
	ResourceName string
}

type CodeRepoPermission struct {
	Projects  []string
	IsProduct bool
}

type CodeRepoTotalPermission struct {
	CodeRepoName string
	ReadWrite    *CodeRepoPermission
	ReadOnly     *CodeRepoPermission
}

type RescourceInformation struct {
	ProductName       string
	Method            string
	ParentResouceKind string
	ParentResouceName string
	ResourceKind      string
	ResourceName      string
}

// SetResourceContext is mount resource information to context.
func SetResourceContext(ctx context.Context, info *RescourceInformation) context.Context {
	return context.WithValue(ctx, ResourceInfoKey, info)
}

func getCodeRepoResourceName(id int) string {
	return fmt.Sprintf("%s%d", RepoPrefix, id)
}

func ConvertCodeRepoToRepoName(ctx context.Context, codeRepo CodeRepo, codeRepoName string) (string, error) {
	id, err := utilstrings.ExtractNumber(RepoPrefix, codeRepoName)
	if err != nil {
		return "", fmt.Errorf("the codeRepo name %s is illegal and must be in the format 'repo-ID'", codeRepoName)
	}

	project, err := codeRepo.GetCodeRepo(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to convert codeRepo name %s to repository name, err: %v", codeRepoName, err)
	}

	return project.Name, nil
}

func ConvertRepoNameToCodeRepoName(ctx context.Context, codeRepo CodeRepo, productName, codeRepoName string) (string, error) {
	pid := fmt.Sprintf("%s/%s", productName, codeRepoName)
	project, err := codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return "", fmt.Errorf("failed to convert repository name %s to codeRepo name, err: %v", codeRepoName, err)
	}

	return fmt.Sprintf("%s%d", RepoPrefix, int(project.ID)), nil
}

func ConvertProductToGroupName(ctx context.Context, codeRepo CodeRepo, productName string) (string, error) {
	var err error
	var id int

	id, err = utilstrings.ExtractNumber(ProductPrefix, productName)
	if err != nil {
		return "", err
	}

	group, err := codeRepo.GetGroup(ctx, id)
	if err != nil {
		return "", err
	}

	return group.Name, nil
}

func ConvertGroupToProductName(ctx context.Context, codeRepo CodeRepo, productName string) (string, error) {
	group, err := codeRepo.GetGroup(ctx, productName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%d", ProductPrefix, int(group.ID)), nil
}

func (p *ProjectPipelineRuntimeUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	var resourceNode *nodestree.Node

	val, ok := data.(*ProjectPipelineRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to save project when create specify node path: %s", path)
	}

	runtime := &resourcev1alpha1.ProjectPipelineRuntime{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.ProjectPipelineRuntime,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	storageResourceDirectory := fmt.Sprintf("%s/%s", path, ProjectsDir)
	resourceParentDir := fmt.Sprintf("%s/%s", storageResourceDirectory, val.Spec.Project)
	resourceFile := fmt.Sprintf("%s/%s.yaml", resourceParentDir, val.Name)
	resourceNode = &nodestree.Node{
		Name:    val.Name,
		Path:    resourceFile,
		Content: runtime,
		Kind:    nodestree.ProjectPipelineRuntime,
		Level:   4,
	}

	return resourceNode, nil
}

func (c *CodeRepoUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*CodeRepoData)
	if !ok {
		return nil, fmt.Errorf("failed to creating specify node, the path: %s", path)
	}

	if val.Spec.Webhook != nil && val.Spec.Webhook.Events == nil {
		val.Spec.Webhook.Events = make([]string, 0)
	}

	codeRepo := &resourcev1alpha1.CodeRepo{
		TypeMeta: v1.TypeMeta{
			Kind:       nodestree.CodeRepo,
			APIVersion: resourcev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, "code-repos")
	resourcePath := fmt.Sprintf("%s/%s/%s.yaml", resourceDirectory, val.Name, val.Name)

	codeRepoProvider, err := getCodeRepoProvider(c.client, c.config.Nautes.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get the information of the codeRepoProvider list when creating node")
	}

	if codeRepoProvider != nil {
		codeRepo.Spec.CodeRepoProvider = codeRepoProvider.Name
	}

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourcePath,
		Content: codeRepo,
		Kind:    nodestree.CodeRepo,
		Level:   4,
	}, nil
}

func (d *DeploymentRuntimeUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*DeploymentRuntimeData)
	if !ok {
		return nil, fmt.Errorf("failed to create node, the path is %s", path)
	}

	resource := &resourcev1alpha1.DeploymentRuntime{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.DeploymentRuntime,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, RuntimesDir)
	resourceFile := fmt.Sprintf("%s/%s.yaml", resourceDirectory, val.Name)

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourceFile,
		Kind:    nodestree.DeploymentRuntime,
		Content: resource,
		Level:   3,
	}, nil
}

func (p *ProjectUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	var resourceNode *nodestree.Node

	val, ok := data.(*ProjectData)
	if !ok {
		return nil, fmt.Errorf("failed to save project when create specify node path: %s", path)
	}

	projectName := val.ProjectName
	productName := val.ProductName
	language := val.Language
	project := &resourcev1alpha1.Project{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.Project,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: projectName,
		},
		Spec: resourcev1alpha1.ProjectSpec{
			Product:  productName,
			Language: language,
		},
	}

	storageResourceDirectory := fmt.Sprintf("%v/%v", path, ProjectsDir)
	resourceParentDir := fmt.Sprintf("%v/%v", storageResourceDirectory, projectName)
	resourceFile := fmt.Sprintf("%v/%v.yaml", resourceParentDir, projectName)
	resourceNode = &nodestree.Node{
		Name:    projectName,
		Path:    resourceFile,
		Content: project,
		Kind:    nodestree.Project,
		Level:   4,
		IsDir:   false,
	}

	return resourceNode, nil
}

func (c *CodeRepoBindingUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*CodeRepoBindingData)
	if !ok {
		return nil, fmt.Errorf("failed to creating specify node, the path: %s", path)
	}

	if len(val.Spec.Projects) == 0 {
		val.Spec.Projects = make([]string, 0)
	}

	codeRepo := &resourcev1alpha1.CodeRepoBinding{
		TypeMeta: v1.TypeMeta{
			Kind:       nodestree.CodeRepoBinding,
			APIVersion: resourcev1alpha1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, CodeReposSubDir)
	resourcePath := fmt.Sprintf("%s/%s/%s.yaml", resourceDirectory, val.Spec.CodeRepo, val.Name)

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourcePath,
		Content: codeRepo,
		Kind:    nodestree.CodeRepoBinding,
		Level:   4,
	}, nil
}

func (e *EnvironmentUsecase) CreateNode(path string, data interface{}) (*nodestree.Node, error) {
	val, ok := data.(*EnviromentData)
	if !ok {
		return nil, errors.New(503, enviromentv1.ErrorReason_ASSERT_ERROR.String(), fmt.Sprintf("failed to assert EnviromentData when create node, data: %v", val))
	}

	env := &resourcev1alpha1.Environment{
		TypeMeta: v1.TypeMeta{
			APIVersion: resourcev1alpha1.GroupVersion.String(),
			Kind:       nodestree.Environment,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: val.Name,
		},
		Spec: val.Spec,
	}

	resourceDirectory := fmt.Sprintf("%s/%s", path, EnvSubDir)
	resourceFile := fmt.Sprintf("%s/%s.yaml", resourceDirectory, val.Name)

	return &nodestree.Node{
		Name:    val.Name,
		Path:    resourceFile,
		Content: env,
		Kind:    nodestree.Environment,
		Level:   3,
	}, nil
}
