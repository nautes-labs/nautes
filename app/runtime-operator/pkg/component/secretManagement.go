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

package component

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

// NewSecretManagement will return an implementation of the secret management.
// Parameter opt indicates the user-defined component option.
// Parameter info indicates the information that can help the secret management to finish operations.
type NewSecretManagement func(opt v1alpha1.Component, info *ComponentInitInfo) (SecretManagement, error)

// SecretManagement provides access management to the secrets.
// It provides access to the secrets requiring an authenticated machine account and authorization for the machine account.
// The secret is stored in the secret store.
// Machine account need to be authenticated to access the secret store. There are various authentication methods. The interface implementer selects the authentication mode.
// Access secrets have permission policies. Users can access secrets only after being authorized to specify permissions.
type SecretManagement interface {
	// Component will be called by the syncer to run generic operations.
	Component

	// GetAccessInfo returns the access information of the cluster, and the cluster identification comes from the NewSecretManagement initialization components information.
	// The cluster identification is the cluster name after the instance of the SecretManagement is created.
	GetAccessInfo(ctx context.Context) (string, error)

	// CreateAccount creates a machine account in the secret store based on the requested machine account information.
	// Additional notes:
	// - The interface implementation should automatically select an authentication mode according to the cluster information,
	//   account information and its own situation, generate authentication information and return.
	// - This interface can be invoked repeatedly. If the machine account information changes,
	//   the authentication information in the secret store needs to be updated.
	// - The cluster information is available from info.ClusterConnectInfo in the new method.
	// - Machine accounts are differentiated by cluster.
	// Return:
	// - In case an error is returned, AuthInfo should be nil.
	CreateAccount(ctx context.Context, account MachineAccount) (*AuthInfo, error)
	// DeleteAccount deletes the machine account from the secret store and clear the secret access permission for this machine account.
	DeleteAccount(ctx context.Context, account MachineAccount) error
	// GrantPermission grants the permission for the machine account to access the specified secret.
	GrantPermission(ctx context.Context, repo SecretInfo, account MachineAccount) error
	// RevokePermission revokes the permission for the machine account to access the specified secret.
	RevokePermission(ctx context.Context, repo SecretInfo, account MachineAccount) error
}

// SecretType represents the type of secret.
type SecretType string

const (
	// SecretTypeCodeRepo means the secret of the code repo type.
	SecretTypeCodeRepo = "CodeRepo"
	// SecretTypeArtifactRepo means the secret of the artifact repo type.
	SecretTypeArtifactRepo = "ArtifactRepo"
)

// SecretInfo represents details information of the secret request that depends on the value of the secret type.
type SecretInfo struct {
	// SecretType represents the type of secret.
	Type SecretType `json:"type"`
	// CodeRepo represents the secret request for the code repo type, if the value of the secret type is code repo then the value is not null.
	CodeRepo *CodeRepo `json:"codeRepo,omitempty"`
	// ArtifactAccount represents the secret request for the artifact repo type, if the value of the secret type is artifact repo then the value is not null.
	ArtifactAccount *ArtifactAccount `json:"artifactAccount,omitempty"`
}

type CodeRepoPermission string

// These are the valid types of permission.
const (
	// CodeRepoPermissionReadOnly means that the read-only permission of the code repo.
	CodeRepoPermissionReadOnly = "readonly"
	// CodeRepoPermissionReadWrite means that the read-write permission of the code repo.
	CodeRepoPermissionReadWrite = "readwrite"
	// CodeRepoPermissionAccessToken means that the access token of the code repo.
	CodeRepoPermissionAccessToken = "accesstoken-api"
)

// CodeRepo represents details information of the secret request for the code repo type.
type CodeRepo struct {
	// ProviderType is type of the code repo provider, such as GitLab or GitHub.
	ProviderType string `json:"providerType,omitempty"`
	// ID is the identity for the Nautes code repo, eg: repo-19.
	ID string
	// User is an identity that can access the code repo, eg: "default".
	User string
	// Permission represents the different access permission of the code repo.
	Permission CodeRepoPermission
}

// ArtifactAccount represents details information of the secret request for the artifact repo type.
type ArtifactAccount struct {
	// ProviderName represents the name of the provided artifact repo.
	ProviderName string
	// Product represents an access identity of the artifact repo is product.
	Product string
	// Project represents an access identity of the artifact repo is project.
	Project string
}
