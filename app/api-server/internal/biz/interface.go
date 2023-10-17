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

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type CodeRepo interface {
	GetCurrentUser(ctx context.Context) (user string, email string, err error)
	CreateCodeRepo(ctx context.Context, gid int, options *GitCodeRepoOptions) (*Project, error)
	UpdateCodeRepo(ctx context.Context, pid interface{}, options *GitCodeRepoOptions) (*Project, error)
	DeleteCodeRepo(ctx context.Context, pid interface{}) error
	GetCodeRepo(ctx context.Context, pid interface{}) (*Project, error)
	ListDeployKeys(ctx context.Context, pid interface{}, opt *ListOptions) ([]*ProjectDeployKey, error)
	ListAllDeployKeys(ctx context.Context, opt *ListOptions) ([]*ProjectDeployKey, error)
	GetDeployKey(ctx context.Context, pid interface{}, deployKeyID int) (*ProjectDeployKey, error)
	EnableProjectDeployKey(ctx context.Context, pid interface{}, deployKey int) (*ProjectDeployKey, error)
	UpdateDeployKey(ctx context.Context, pid interface{}, deployKey int, title string, canPush bool) (*ProjectDeployKey, error)
	SaveDeployKey(ctx context.Context, pid interface{}, title string, canPush bool, publicKey []byte) (*ProjectDeployKey, error)
	DeleteDeployKey(ctx context.Context, pid interface{}, deployKey int) error
	CreateGroup(ctx context.Context, gitOptions *GitGroupOptions) (*Group, error)
	DeleteGroup(ctx context.Context, gid interface{}) error
	UpdateGroup(ctx context.Context, gid interface{}, git *GitGroupOptions) (*Group, error)
	GetGroup(ctx context.Context, gid interface{}) (*Group, error)
	ListAllGroups(ctx context.Context) ([]*Group, error)
	ListGroupCodeRepos(ctx context.Context, gid interface{}, opts *ListGroupProjectsOptions) ([]*Project, error)
	ListCodeRepos(ctx context.Context, search string) ([]*Project, error)
	ListAccessTokens(ctx context.Context, pid interface{}, opt *ListOptions) ([]*ProjectAccessToken, error)
	GetProjectAccessToken(ctx context.Context, pid interface{}, id int) (*ProjectAccessToken, error)
	CreateProjectAccessToken(ctx context.Context, pid interface{}, opt *CreateProjectAccessTokenOptions) (*ProjectAccessToken, error)
	DeleteProjectAccessToken(ctx context.Context, pid interface{}, id int) error
}

type Secretrepo interface {
	GetSecret(ctx context.Context, secretOptions *SecretOptions) (string, error)
	GetDeployKey(ctx context.Context, secretOptions *SecretOptions) (*DeployKeySecretData, error)
	SaveDeployKey(ctx context.Context, data *SecretData, extendKVs map[string]string) error
	SaveProjectAccessToken(ctx context.Context, id, token, user, permission string, extendKVs map[string]string) error
	SaveClusterConfig(ctx context.Context, id, config string) error
	DeleteSecret(ctx context.Context, id int, user, permission string) error
	AuthorizationSecret(ctx context.Context, id int, destUser, gitType, mountPath string) error
	GetProjectAccessToken(ctx context.Context, secretOptions *SecretOptions) (*AccessTokenSecretData, error)
}

type GitRepo interface {
	Commit(ctx context.Context, path string) error
	SaveConfig(ctx context.Context, path string) error
	Clone(ctx context.Context, param *CloneRepositoryParam) (string, error)
	Merge(ctx context.Context, path string) (string, error)
	Push(ctx context.Context, path string, command ...string) error
	Diff(ctx context.Context, path string, command ...string) (string, error)
	Fetch(ctx context.Context, path string, command ...string) (string, error)
}

type DexRepo interface {
	UpdateRedirectURIs(redirectURI string) error
	RemoveRedirectURIs(redirectURIs string) error
}

type Kubernetes interface {
	ListCodeRepo(ctx context.Context) (*resourcev1alpha1.CodeRepoList, error)
	ListCodeRepoBindings(ctx context.Context) (*resourcev1alpha1.CodeRepoBindingList, error)
}
