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

// NewMultiTenant will return a MultiTenant implementation.
// opt is the user defined component options.
// info is the information can help MultiTenant to finish jobs.
type NewMultiTenant func(opt v1alpha1.Component, info *ComponentInitInfo) (MultiTenant, error)

// MultiTenant manages the environment and machine account information in the cluster.
// It mainly has three concepts:
//
//   - Product:
//
// The product is responsible for recording the environment and machine account information of the nautes product in the cluster.
//
//   - Space:
//     Space is a logical independent space where user deploys programs and executes pipeline tasks.
//     Depending on the cluster type, the final space created will be different:
//     -- for a namespace in the Kubernetes cluster,
//     -- In the server cluster, for an independent folder (each machine in the cluster occupies this space.)
//
//   - MachineAccount:
//     MachineAccount is the machine account that ensures the normal operation of the runtime.
//     -- It can operate on all resources in space.
//     -- The program can run in the environment through the identity of this account.
//     -- Components can obtain authentication information from remote systems through it.
//     -- Components can use it to operate on the permissions of related resources in the remote system.
//
//     The account name is unique in the cluster.
type MultiTenant interface {
	Component

	// CreateProduct will create a product in the cluster based on name.
	//
	// Return:
	// - If the product already exists, return nil directly.
	CreateProduct(ctx context.Context, name string) error

	// DeleteProduct will delete the product in the cluster based on the name.
	//
	// Return:
	// - If the product does not exist, return nil.
	// - If there are still spaces and machine accounts under the product name, the deletion fails.
	DeleteProduct(ctx context.Context, name string) error

	// CreateSpace will divide a dedicated space in the cluster.
	//
	// Additional notes:
	// - The interface can be executed repeatedly.
	//
	// Return:
	// - If the product does not exist, use the 'ProductNotFound' method to return an error.
	CreateSpace(ctx context.Context, productName string, name string) error

	// DeleteSpace will delete the dedicated area created in the cluster
	//
	// Additional notes:
	// - Ignoring the usage status of space when deleting.
	//
	// Return:
	// - If the product or space does not exist, return nil.
	DeleteSpace(ctx context.Context, productName string, name string) error

	// GetSpace returns detailed information about the space based on the product name and name.
	//
	// Return:
	// - If the product does not exist, use the 'ProductNotFound' method to return an error.
	// - If space does not exist, return an error.
	GetSpace(ctx context.Context, productName, name string) (*SpaceStatus, error)

	// ListSpaces returns detailed information about all spaces under the specified product.
	//
	// Return:
	// - If the product does not exist, use the 'ProductNotFound' method to return an error.
	// - If the search result is empty, return an empty array.
	ListSpaces(ctx context.Context, productName string) ([]SpaceStatus, error)

	// AddSpaceUser will authorize the user or machine account to have management permissions for the space.
	//
	// Additional note:
	// - When the request scope is 'RequestScopeProduct', the authorization object is a role.request.User is the product name
	// - When the request scope is 'RequestScopeProject', the authorization object is a role.request.User is the project name.
	// - When the request scope is 'RequestScopeAccount', the authorized object is the machine account in the product.request.User is the account name.
	// - Here are two examples of how to give machine accounts administrative privileges for spaces:
	//     -- When the cluster type is kubernetes, create a service account under the namespace corresponding to the space, and bind the sa to the role 'cluster-admin'.
	//     -- When the cluster type is server and the system is Linux, create a group specifically for the space and add the account to the group.
	//
	// Return:
	// - If space does not exist, return an error.
	// - When 'request.User' does not exist, return an error.
	AddSpaceUser(ctx context.Context, request PermissionRequest) error

	// DeleteSpaceUser revokes the user or machine account's management permissions on the space.
	//
	// Return:
	// - If space does not exist, return nil.
	// - When 'request.User' does not exist, return nil.
	DeleteSpaceUser(ctx context.Context, request PermissionRequest) error

	// CreateAccount will create a machine account in the specified product based on name.
	//
	// Additional notes:
	// - Other components should not call this method.
	//
	// Return:
	// - Return an error when name is already used by another product.
	// - When the product does not exist, use the 'ProductNotFound' method to return an error.
	CreateAccount(ctx context.Context, productName, name string) error

	// DeleteAccount will delete the specified machine account under the product.
	//
	// Additional note:
	// - When deleting, ignore whether the machine account is still being used by other components.
	// - When deleting, ignore whether the authentication information in the remote system has been cleared.
	// - The resources referenced by the account and the authentication information of the account in the remote system are cleaned up by the runtime operator.
	// - Other components should not call this method.
	//
	// Return:
	// - When the Product or account does not exist, return nil.
	DeleteAccount(ctx context.Context, productName, name string) error

	// GetAccount returns detailed information about the machine account.
	//
	// Return:
	// - When the product or account does not exist, use the 'AccountNotFound' method to return an error.
	GetAccount(ctx context.Context, productName, name string) (*MachineAccount, error)
}
