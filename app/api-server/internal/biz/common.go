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

	utilstrings "github.com/nautes-labs/nautes/app/api-server/util/string"
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

func ConvertGroupToProductName(ctx context.Context, codeRepo CodeRepo, productName string) (string, error) {
	group, err := codeRepo.GetGroup(ctx, productName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%d", ProductPrefix, int(group.ID)), nil
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

func ConvertRepoNameToCodeRepoName(ctx context.Context, codeRepo CodeRepo, productName, codeRepoName string) (string, error) {
	pid := fmt.Sprintf("%s/%s", productName, codeRepoName)
	project, err := codeRepo.GetCodeRepo(ctx, pid)
	if err != nil {
		return "", fmt.Errorf("failed to convert repository name %s to codeRepo name, err: %v", codeRepoName, err)
	}

	return fmt.Sprintf("%s%d", RepoPrefix, int(project.ID)), nil
}
