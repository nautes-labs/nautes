// Copyright 2024 Nautes Authors
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

package componentmock

import (
	"context"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
)

// sm *SecretManagement component.SecretManagement
type SecretManagement struct {
	AccessInfo string
}

// CleanUp is used by components to refresh and clean data.
// Runtime operator will call this method at the end of reconcile.
// The error returned will not cause reconcile failure, it will only be output to the log.
func (sm *SecretManagement) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

// GetComponentMachineAccount returns the machine account used by the component, and the runtime operator will give the account the resource access it should have.
// If the machine account is not needed, return nil.
func (sm *SecretManagement) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

// GetAccessInfo returns the access information of the cluster, and the cluster identification comes from the NewSecretManagement initialization components information.
// The cluster identification is the cluster name after the instance of the SecretManagement is created.
func (sm *SecretManagement) GetAccessInfo(_ context.Context) (string, error) {
	return sm.AccessInfo, nil
}

// CreateAccount creates a machine account in the secret store based on the requested machine account information.
// Additional notes:
//   - The interface implementation should automatically select an authentication mode according to the cluster information,
//     account information and its own situation, generate authentication information and return.
//   - This interface can be invoked repeatedly. If the machine account information changes,
//     the authentication information in the secret store needs to be updated.
//   - The cluster information is available from info.ClusterConnectInfo in the new method.
//   - Machine accounts are differentiated by cluster.
//
// Return:
// - In case an error is returned, AuthInfo should be nil.
func (sm *SecretManagement) CreateAccount(ctx context.Context, account component.MachineAccount) (*component.AuthInfo, error) {
	panic("not implemented") // TODO: Implement
}

// DeleteAccount deletes the machine account from the secret store and clear the secret access permission for this machine account.
func (sm *SecretManagement) DeleteAccount(ctx context.Context, account component.MachineAccount) error {
	panic("not implemented") // TODO: Implement
}

// GrantPermission grants the permission for the machine account to access the specified secret.
func (sm *SecretManagement) GrantPermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	panic("not implemented") // TODO: Implement
}

// RevokePermission revokes the permission for the machine account to access the specified secret.
func (sm *SecretManagement) RevokePermission(ctx context.Context, repo component.SecretInfo, account component.MachineAccount) error {
	panic("not implemented") // TODO: Implement
}
