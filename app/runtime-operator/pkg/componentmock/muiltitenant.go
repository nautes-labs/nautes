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
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	"k8s.io/apimachinery/pkg/util/sets"
)

// mt *MultiTenant component.MultiTenant
type MultiTenant struct {
	SpaceType component.SpaceType
	Products  map[string]Product
}

type Product struct {
	Name             string
	Spaces           sets.Set[string]
	Accounts         sets.Set[string]
	SpaceUserBinding []SpaceUserBinding
}

type SpaceUserBinding struct {
	Space string
	User  string
}

func (mt *MultiTenant) Init() error {
	mt.Products = map[string]Product{}
	if mt.SpaceType == "" {
		mt.SpaceType = component.SpaceTypeKubernetes
	}
	return nil
}

// CleanUp is used by components to refresh and clean data.
// Runtime operator will call this method at the end of reconcile.
// The error returned will not cause reconcile failure, it will only be output to the log.
func (mt *MultiTenant) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

// GetComponentMachineAccount returns the machine account used by the component, and the runtime operator will give the account the resource access it should have.
// If the machine account is not needed, return nil.
func (mt *MultiTenant) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

// CreateProduct will create a product in the cluster based on name.
//
// Return:
// - If the product already exists, return nil directly.
func (mt *MultiTenant) CreateProduct(ctx context.Context, name string) error {
	if _, ok := mt.Products[name]; ok {
		return nil
	}

	mt.Products[name] = Product{
		Name:             name,
		Spaces:           sets.New[string](),
		Accounts:         sets.New[string](),
		SpaceUserBinding: []SpaceUserBinding{},
	}

	mt.CreateSpace(ctx, name, name)

	return nil
}

// DeleteProduct will delete the product in the cluster based on the name.
//
// Return:
// - If the product does not exist, return nil.
// - If there are still spaces and machine accounts under the product name, the deletion fails.
func (mt *MultiTenant) DeleteProduct(ctx context.Context, name string) error {
	if _, ok := mt.Products[name]; !ok {
		return nil
	}

	if len(mt.Products[name].Spaces) > 1 || len(mt.Products[name].Accounts) > 0 {
		return fmt.Errorf("product %s still has spaces or accounts", name)
	}

	mt.DeleteSpace(ctx, name, name)

	delete(mt.Products, name)
	return nil
}

// CreateSpace will divide a dedicated space in the cluster.
//
// Additional notes:
// - The interface can be executed repeatedly.
//
// Return:
// - If the product does not exist, use the 'ProductNotFound' method to return an error.
func (mt *MultiTenant) CreateSpace(ctx context.Context, productName string, name string) error {
	if _, ok := mt.Products[productName]; !ok {
		return component.ProductNotFound(nil, productName)
	}

	if mt.Products[productName].Spaces.Has(name) {
		return nil
	}

	mt.Products[productName].Spaces.Insert(name)
	return nil
}

// DeleteSpace will delete the dedicated area created in the cluster
//
// Additional notes:
// - Ignoring the usage status of space when deleting.
//
// Return:
// - If the product or space does not exist, return nil.
func (mt *MultiTenant) DeleteSpace(ctx context.Context, productName string, name string) error {
	if _, ok := mt.Products[productName]; !ok {
		return nil
	}

	if !mt.Products[productName].Spaces.Has(name) {
		return nil
	}

	mt.Products[productName].Spaces.Delete(name)
	return nil
}

// GetSpace returns detailed information about the space based on the product name and name.
//
// Return:
// - If the product does not exist, use the 'ProductNotFound' method to return an error.
// - If space does not exist, return an error.
func (mt *MultiTenant) GetSpace(ctx context.Context, productName string, name string) (*component.SpaceStatus, error) {
	if _, ok := mt.Products[productName]; !ok {
		return nil, component.ProductNotFound(nil, productName)
	}

	if !mt.Products[productName].Spaces.Has(name) {
		return nil, fmt.Errorf("space %s not found", name)
	}

	accounts := []string{}
	for _, binding := range mt.Products[productName].SpaceUserBinding {
		if binding.Space == name {
			accounts = append(accounts, binding.User)
		}
	}

	return &component.SpaceStatus{
		Space: component.Space{
			ResourceMetaData: component.ResourceMetaData{
				Product: productName,
				Name:    name,
			},
			SpaceType: mt.SpaceType,
		},
		Accounts: accounts,
	}, nil

}

// ListSpaces returns detailed information about all spaces under the specified product.
//
// Return:
// - If the product does not exist, use the 'ProductNotFound' method to return an error.
// - If the search result is empty, return an empty array.
func (mt *MultiTenant) ListSpaces(ctx context.Context, productName string) ([]component.SpaceStatus, error) {
	if _, ok := mt.Products[productName]; !ok {
		return nil, component.ProductNotFound(nil, productName)
	}

	spaces := []component.SpaceStatus{}
	for _, space := range mt.Products[productName].Spaces.UnsortedList() {
		status, err := mt.GetSpace(ctx, productName, space)
		if err != nil {
			return nil, err
		}
		spaces = append(spaces, *status)
	}

	return spaces, nil
}

// AddSpaceUser will authorize the user or machine account to have management permissions for the space.
//
// Additional note:
//   - When the request scope is 'RequestScopeProduct', the authorization object is a role.request.User is the product name
//   - When the request scope is 'RequestScopeProject', the authorization object is a role.request.User is the project name.
//   - When the request scope is 'RequestScopeAccount', the authorized object is the machine account in the product.request.User is the account name.
//   - Here are two examples of how to give machine accounts administrative privileges for spaces:
//     -- When the cluster type is kubernetes, create a service account under the namespace corresponding to the space, and bind the sa to the role 'cluster-admin'.
//     -- When the cluster type is server and the system is Linux, create a group specifically for the space and add the account to the group.
//
// Return:
// - If space does not exist, return an error.
// - When 'request.User' does not exist, return an error.
func (mt *MultiTenant) AddSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	productName := request.Resource.Product
	if _, ok := mt.Products[productName]; !ok {
		return component.ProductNotFound(nil, productName)
	}

	product := mt.Products[productName]

	if !product.Spaces.Has(request.Resource.Name) {
		return fmt.Errorf("space %s not found", request.Resource.Name)
	}

	if !product.Accounts.Has(request.User) {
		return fmt.Errorf("account %s not found", request.User)
	}

	for _, binding := range product.SpaceUserBinding {
		if binding.Space == request.Resource.Name && binding.User == request.User {
			return nil
		}
	}

	product.SpaceUserBinding = append(product.SpaceUserBinding, SpaceUserBinding{
		Space: request.Resource.Name,
		User:  request.User,
	})

	mt.Products[productName] = product
	return nil
}

// DeleteSpaceUser revokes the user or machine account's management permissions on the space.
//
// Return:
// - If space does not exist, return nil.
// - When 'request.User' does not exist, return nil.
func (mt *MultiTenant) DeleteSpaceUser(ctx context.Context, request component.PermissionRequest) error {
	productName := request.Resource.Product
	if _, ok := mt.Products[productName]; !ok {
		return component.ProductNotFound(nil, productName)
	}

	product := mt.Products[productName]

	if !product.Spaces.Has(request.Resource.Name) {
		return nil
	}

	if !product.Accounts.Has(request.User) {
		return nil
	}

	for i, binding := range product.SpaceUserBinding {
		if binding.Space == request.Resource.Name && binding.User == request.User {
			product.SpaceUserBinding = append(product.SpaceUserBinding[:i], product.SpaceUserBinding[i+1:]...)
			break
		}
	}

	mt.Products[productName] = product
	return nil
}

// CreateAccount will create a machine account in the specified product based on name.
//
// Additional notes:
// - Other components should not call this method.
//
// Return:
// - Return an error when name is already used by another product.
// - When the product does not exist, use the 'ProductNotFound' method to return an error.
func (mt *MultiTenant) CreateAccount(ctx context.Context, productName string, name string) error {
	if _, ok := mt.Products[productName]; !ok {
		return component.ProductNotFound(nil, productName)
	}

	for _, product := range mt.Products {
		if product.Accounts.Has(name) {
			if product.Name == productName {
				return nil
			}
			return fmt.Errorf("account %s already exists", name)
		}
	}

	mt.Products[productName].Accounts.Insert(name)
	return nil
}

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
func (mt *MultiTenant) DeleteAccount(ctx context.Context, productName string, name string) error {
	if _, ok := mt.Products[productName]; !ok {
		return nil
	}

	if !mt.Products[productName].Accounts.Has(name) {
		return nil
	}

	mt.Products[productName].Accounts.Delete(name)
	return nil
}

// GetAccount returns detailed information about the machine account.
//
// Return:
// - When the product or account does not exist, use the 'AccountNotFound' method to return an error.
func (mt *MultiTenant) GetAccount(ctx context.Context, productName string, name string) (*component.MachineAccount, error) {
	if _, ok := mt.Products[productName]; !ok {
		return nil, component.ProductNotFound(nil, productName)
	}

	if !mt.Products[productName].Accounts.Has(name) {
		return nil, component.AccountNotFound(nil, name)
	}

	spaces := []string{}
	for _, binding := range mt.Products[productName].SpaceUserBinding {
		if binding.User == name {
			spaces = append(spaces, binding.Space)
		}
	}

	return &component.MachineAccount{
		Name:    name,
		Product: productName,
		Spaces:  spaces,
	}, nil
}
