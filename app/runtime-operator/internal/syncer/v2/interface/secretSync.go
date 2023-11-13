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

// NewSecretSync will return an implementation of the secret synchronization.
// Parameter opt indicates the user-defined component option.
// Parameter info indicates the information that can help the secret synchronization to finish operations.
type NewSecretSync func(opt v1alpha1.Component, info *ComponentInitInfo) (SecretSync, error)

// SecretSync provides secret synchronization function to aim the secrets from the secret store sync to the destination environment.
// The secret is stored in the secret store.
// By creating a secret synchronization object in the destination environment.
// Synchronization operation is performed by the secret synchronization object.
// Finally, the synchronization result of the secret is generated.
type SecretSync interface {
	// Component will be called by the syncer to run generic operations.
	Component

	// CreateSecret creates the secret synchronization object, the synchronization result of the secret depends on the space type in the destination environment
	// Space type is the "SecretRequest.Destination.Space.SpaceType".
	// If the space type of the request is Kubernetes then is a Secret name else is a folder name on Host.
	CreateSecret(ctx context.Context, secretReq SecretRequest) error
	// RemoveSecret removes the secret synchronization object from the dest environment of the request.
	RemoveSecret(ctx context.Context, secretReq SecretRequest) error
}

// SecretRequest represents the details of the secret request information, user authentication information, and target environment.
type SecretRequest struct {
	// Name is the identity of the request.
	Name string
	// Source represents details information of request secret.
	Source SecretInfo
	// AuthInfo represents the authentication information of the machine account.
	AuthInfo *AuthInfo
	// Destination represents the destination environment that is used to save secret.
	Destination SecretRequestDestination
}

// SecretRequestDestination providers the destination environment that is used to save secret.
type SecretRequestDestination struct {
	// Name is a secret name in the destination environment.
	Name string
	// Space represents a space to store secret in the destination environment, eg: the namespace of Kubernetes.
	Space Space
	// Format represents finally secret info format.
	Format string
}
