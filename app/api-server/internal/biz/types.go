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

import "time"

type GroupOptions struct {
	// The name of the group
	Name string `json:"name,omitempty"`
	// The path of the group
	Path string `json:"path,omitempty"`
	// The visibility level of the group
	Visibility string `json:"visibility,omitempty"`
	// The description of the group
	Description string `json:"description,omitempty"`
	// The ID of the parent group
	ParentId int32 `json:"parent_id,omitempty"`
}

type GitGroupOptions struct {
	Github *GroupOptions
	Gitlab *GroupOptions
}

type GitlabCodeRepoOptions struct {
	// The name of the repository
	Name string `json:"name,omitempty"`
	// The path of the repository
	Path string `json:"path,omitempty"`
	// The visibility level of the repository
	Visibility string `json:"visibility,omitempty"`
	// The description of the repository
	Description string `json:"description,omitempty"`
	// The ID of the namespace to which the repository belongs
	NamespaceID int32 `json:"namespace_id,omitempty"`
}

type GitCodeRepoOptions struct {
	Gitlab *GitlabCodeRepoOptions
	Github interface{}
}

type CloneRepositoryParam struct {
	// The URL of the repository to clone
	URL string
	// The user to use for the clone operation
	User string
	// The email to use for the clone operation
	Email string
}

type SecretData struct {
	// The ID of the secret
	ID int `json:"id"`
	// The data associated with the secret
	Data string `json:"key"`
}

type ProjectDeployKey struct {
	// The ID of the deploy key
	ID int `json:"id"`
	// The title of the deploy key
	Title string `json:"title"`
	// The key value of the deploy key
	Key string `json:"key"`
	// The date and time when the deploy key was created
	CreatedAt *time.Time `json:"created_at"`
	// Whether the deploy key has push access
	CanPush bool `json:"can_push"`
	// The fingerprint of the deploy key
	Fingerprint string `json:"fingerprint"`
	// The projects that have write access to this deploy key
	ProjectsWithWriteAccess []*DeployKeyProject `json:"projects_with_write_access"`
}

type DeployKeyProject struct {
	// The ID of the project
	ID int `json:"id"`
	// The description of the project
	Description string `json:"description"`
	// The name of the project
	Name string `json:"name"`
	// The name of the project with namespace
	NameWithNamespace string `json:"name_with_namespace"`
	// The path of the project
	Path string `json:"path"`
	// The path of the project with namespace
	PathWithNamespace string `json:"path_with_namespace"`
	// The date and time when the project was created
	CreatedAt *time.Time `json:"created_at"`
}

type SecretOptions struct {
	// The path of the secret
	SecretPath string
	// The engine used to store the secret
	SecretEngine string
	// The key used to access the secret
	SecretKey string
}

type DeployKeyType string
type DeployKeySecretData struct {
	// The ID of the deploy key secret
	ID int
	// The fingerprint associated with the deploy key secret
	Fingerprint string
}

type AccessTokenSecretData struct {
	// The ID of the access token secret
	ID int
}

type ListOptions struct {
	// The page number
	Page int `url:"page,omitempty" json:"page,omitempty"`
	// The number of items per page
	PerPage int `url:"per_page,omitempty" json:"per_page,omitempty"`
}

type ListGroupProjectsOptions struct {
	ListOptions
	Search string `url:"search,omitempty" json:"search,omitempty"`
	Sort   string `url:"sort,omitempty" json:"sort,omitempty"`
}

// Project Access token type definition.
type ISOTime time.Time

type AccessLevelValue int
type AccessTokenPermission string

type ProjectAccessToken struct {
	// ID represents the unique identifier of the project access token.
	ID int `json:"id"`
	// UserID represents the unique identifier of the user associated with the project access token.
	UserID int `json:"user_id"`
	// Name represents the name of the project access token.
	Name string `json:"name"`
	// Scopes represents the scopes granted to the project access token.
	Scopes []string `json:"scopes"`
	// CreatedAt represents the date and time when the project access token was created.
	CreatedAt *time.Time `json:"created_at"`
	// LastUsedAt represents the date and time when the project access token was last used.
	LastUsedAt *time.Time `json:"last_used_at"`
	// ExpiresAt represents the date and time when the project access token will expire.
	ExpiresAt *ISOTime `json:"expires_at"`
	// Active represents whether the project access token is active or not.
	Active bool `json:"active"`
	// Revoked represents whether the project access token has been revoked or not.
	Revoked bool `json:"revoked"`
	// Token represents the value of the project access token.
	Token string `json:"token"`
	// AccessLevel represents the access level granted to the project access token.
	AccessLevel AccessLevelValue `json:"access_level"`
}

type CreateProjectAccessTokenOptions struct {
	// Name represents the name of the project access token.
	Name *string `url:"name,omitempty" json:"name,omitempty"`
	// Scopes represents the scopes to be granted to the project access token.
	Scopes *[]string `url:"scopes,omitempty" json:"scopes,omitempty"`
	// AccessLevel represents the access level to be granted to the project access token.
	AccessLevel *AccessLevelValue `url:"access_level,omitempty" json:"access_level,omitempty"`
	// ExpiresAt represents the date and time when the project access token will expire.
	ExpiresAt *ISOTime `url:"expires_at,omitempty" json:"expires_at,omitempty"`
}
