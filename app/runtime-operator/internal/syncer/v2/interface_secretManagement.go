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

package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewSecretManagement func(opt v1alpha1.Component, info *ComponentInitInfo) (SecretManagement, error)

type SecretManagement interface {
	Component

	// GetAccessInfo should return the information on how to access the cluster
	GetAccessInfo(ctx context.Context) (string, error)

	CreateUser(ctx context.Context, user User) error
	DeleteUser(ctx context.Context, user User) error
	GrantPermission(ctx context.Context, repo SecretInfo, user User) error
	RevokePermission(ctx context.Context, repo SecretInfo, user User) error
}

type SecretType string

const (
	SecretTypeCodeRepo     = "CodeRepo"
	SecretTypeArtifactRepo = "ArtifactRepo"
)

type SecretInfo struct {
	Type SecretType
	// CodeRepo 类型的密钥
	CodeRepo *CodeRepo
	// Artifact 类型的密钥
	ArtifactAccount *ArtifactAccount
}

type CodeRepoPermission string

const (
	CodeRepoPermissionReadOnly    = "readonly"
	CodeRepoPermissionReadWrite   = "readwrite"
	CodeRepoPermissionAccessToken = "accesstoken-api"
)

type CodeRepo struct {
	ProviderType string
	ID           string
	User         string
	Permission   CodeRepoPermission
}

type ArtifactAccount struct {
	ProviderName string
	Product      string
	Project      string
}
