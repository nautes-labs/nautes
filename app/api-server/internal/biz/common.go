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
	// layout resouce directory names.
	CodeReposSubDir = "code-repos"
	RuntimesDir     = "runtimes"
	EnvSubDir       = "envs"
	ProjectsDir     = "projects"
)

// Constants used to store commit information.
const (
	ResourceInfoKey = "ResourceInfoKey"
	// mark this request method
	SaveMethod   = "Save"
	DeleteMethod = "Delete"
)

// This is inital page options.
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

// SetResourceContext is mount resource information to context.
func SetResourceContext(ctx context.Context, productName, method, parentResouceKind, parentResouceName, resourceKind, resourceName string) context.Context {
	info := &ResourceInfo{
		ProductName:        productName,
		Method:             method,
		ParentResouceKind:  parentResouceKind,
		ParentResourceName: parentResouceName,
		ResourceKind:       resourceKind,
		ResourceName:       resourceName,
	}
	return context.WithValue(ctx, ResourceInfoKey, info)
}

func getCodeRepoResourceName(id int) string {
	return fmt.Sprintf("%s%d", RepoPrefix, id)
}
