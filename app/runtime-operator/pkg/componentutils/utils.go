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

package componentutils

import (
	"context"
	"fmt"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"

	"k8s.io/apimachinery/pkg/util/sets"
)

// CreateOrUpdateProduct creates or updates a product with the given name and namespaces.
// If the product already exists, it will be updated to match the given namespaces.
// If the product does not exist, it will be created with the given namespaces.
// If the product exists but the namespaces are empty, the product will be deleted.
// If the product exists but the namespaces are not empty, the product will be updated to match the given namespaces.
func CreateOrUpdateProduct(ctx context.Context, multiTenant component.MultiTenant, productName string, productNamespaces []string) error {
	err := multiTenant.CreateProduct(ctx, productName)
	if err != nil {
		return fmt.Errorf("failed to create product %s: %w", productName, err)
	}

	oldNamespaceStatuses, err := multiTenant.ListSpaces(ctx, productName)
	if err != nil {
		return fmt.Errorf("failed to list namespace %s: %w", productName, err)
	}

	newNamespaceSet := sets.New(productNamespaces...)
	oldNamespaceSet := sets.New[string]()
	for _, nsStatus := range oldNamespaceStatuses {
		oldNamespaceSet.Insert(nsStatus.Name)
	}

	for _, namespace := range newNamespaceSet.Difference(oldNamespaceSet).UnsortedList() {
		err := multiTenant.CreateSpace(ctx, productName, namespace)
		if err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// DeleteNamespaces deletes the given namespaces from the given product.
func DeleteNamespaces(ctx context.Context, multiTenant component.MultiTenant, productName string, namespaces []string) error {
	for _, namespace := range namespaces {
		err := multiTenant.DeleteSpace(ctx, productName, namespace)
		if err != nil {
			return fmt.Errorf("failed to delete namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// UpdateOrDeleteProduct updates or deletes a product with the given name and namespaces.
// It will delete namespaces that are not in the given namespaces.
// If namespaces is empty after deleting, the product will be deleted.
func DeleteOrUpdateProduct(ctx context.Context, multiTenant component.MultiTenant, productName string, dstNamespaces []string) error {
	space, err := multiTenant.ListSpaces(ctx, productName)
	if err != nil {
		return fmt.Errorf("failed to list namespace %s: %w", productName, err)
	}

	dstNamespaceSet := sets.New(dstNamespaces...)
	currentNamespaceSet := sets.New[string]()
	for _, nsStatus := range space {
		if nsStatus.Name == productName {
			continue
		}
		currentNamespaceSet.Insert(nsStatus.Name)
	}

	deleteNamespaces := currentNamespaceSet.Difference(dstNamespaceSet).UnsortedList()
	if err := DeleteNamespaces(ctx, multiTenant, productName, deleteNamespaces); err != nil {
		return err
	}

	// Check the rest of namespaces, if there is only one namespace left, delete the product.
	if len(space)-len(deleteNamespaces) == 1 {
		if err := multiTenant.DeleteProduct(ctx, productName); err != nil {
			return fmt.Errorf("failed to delete product %s: %w", productName, err)
		}
	}

	return nil
}

// CreateOrUpdateProductAccount creates or updates a product account for a given user.
// It checks if the account already exists and creates a new one if it doesn't.
// It also updates the space permissions for the account based on the provided newSpacePermission.
// If any errors occur during the process, an error is returned.
func CreateOrUpdateProductAccount(ctx context.Context, multiTenant component.MultiTenant, productName, userName string, newSpacePermission []string) error {
	spaceUsage := []string{}
	account, err := multiTenant.GetAccount(ctx, productName, userName)
	if err != nil {
		if !component.IsAccountNotFound(err) {
			return fmt.Errorf("failed to get account %s: %w", userName, err)
		}
		err := multiTenant.CreateAccount(ctx, productName, userName)
		if err != nil {
			return fmt.Errorf("failed to create account %s: %w", userName, err)
		}
	} else {
		spaceUsage = account.Spaces
	}

	newUsageSet := sets.New(newSpacePermission...)
	oldUsageSet := sets.New(spaceUsage...)

	for _, space := range newUsageSet.Difference(oldUsageSet).UnsortedList() {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    space,
			},
			User:         userName,
			RequestScope: component.RequestScopeAccount,
			Permission:   component.Permission{},
		}
		err := multiTenant.AddSpaceUser(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to add user %s to space %s: %w", userName, space, err)
		}
	}

	for _, space := range oldUsageSet.Difference(newUsageSet).UnsortedList() {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    space,
			},
			User:         userName,
			RequestScope: component.RequestScopeAccount,
			Permission:   component.Permission{},
		}
		err := multiTenant.DeleteSpaceUser(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to remove user %s from space %s: %w", userName, space, err)
		}
	}

	return nil
}

// DeleteProductAccount deletes a product account for a given user.
// It checks if the account exists and deletes it if it does.
// If any errors occur during the process, an error is returned.
func DeleteProductAccount(ctx context.Context, multiTenant component.MultiTenant, productName, userName string) error {
	account, err := multiTenant.GetAccount(ctx, productName, userName)
	if err != nil {
		if component.IsAccountNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get account %s: %w", userName, err)
	}

	for _, space := range account.Spaces {
		req := component.PermissionRequest{
			Resource: component.ResourceMetaData{
				Product: productName,
				Name:    space,
			},
			User:         account.Name,
			RequestScope: component.RequestScopeAccount,
			Permission:   component.Permission{},
		}
		err := multiTenant.DeleteSpaceUser(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to remove user %s from space %s: %w", userName, space, err)
		}
	}

	err = multiTenant.DeleteAccount(ctx, productName, userName)
	if err != nil {
		return fmt.Errorf("failed to delete account %s: %w", userName, err)
	}

	return nil
}

// CreateOrUpdateSecretManagementAccount creates or updates a secret management account for a given product and account name.
// It retrieves the account from the multi-tenant component, creates the account in the secret management component,
// and grants permissions to the account for the specified secret information.
// It returns the authentication information for the created or updated account, or an error if any operation fails.
func CreateOrUpdateSecretManagementAccount(
	ctx context.Context,
	multiTenant component.MultiTenant,
	secMgr component.SecretManagement,
	productName, accountName string,
	secInfos []component.SecretInfo,
) (*component.AuthInfo, error) {
	account, err := multiTenant.GetAccount(ctx, productName, accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", accountName, err)
	}

	authInfo, err := secMgr.CreateAccount(ctx, *account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account %s in secret management : %w", account, err)
	}

	for _, secInfo := range secInfos {
		err := secMgr.GrantPermission(ctx, secInfo, *account)
		if err != nil {
			return nil, fmt.Errorf("failed to grant permission to account %s: %w", account, err)
		}
	}

	return authInfo, nil
}

// DeleteSecretManagementAccount deletes a secret management account.
// It retrieves the account using the multiTenant component and the provided product name and account name.
// If the account is not found, it returns nil.
// Otherwise, it deletes the account using the secMgr component.
// Returns an error if any operation fails.
func DeleteSecretManagementAccount(ctx context.Context, multiTenant component.MultiTenant, secMgr component.SecretManagement, productName, accountName string) error {
	account, err := multiTenant.GetAccount(ctx, productName, accountName)
	if err != nil {
		if component.IsAccountNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get account %s: %w", accountName, err)
	}

	err = secMgr.DeleteAccount(ctx, *account)
	if err != nil {
		return fmt.Errorf("failed to delete account %s: %w", account, err)
	}
	return nil
}

// CompareStrings compares two slices of strings and returns three slices:
// - onlyInA: strings that are present in slice a but not in slice b
// - onlyInB: strings that are present in slice b but not in slice a
// - inBoth: strings that are present in both slice a and slice b
func CompareStrings(a, b []string) ([]string, []string, []string) {
	setA := sets.New(a...)
	setB := sets.New(b...)

	onlyInA := setA.Difference(setB).UnsortedList()
	onlyInB := setB.Difference(setA).UnsortedList()
	inBoth := setA.Intersection(setB).UnsortedList()

	return onlyInA, onlyInB, inBoth
}
