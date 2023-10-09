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

package util

import (
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsLegal check resource is ready to used.
// When function a returns an error, it means that this resource should not be used anymore.
// This may be because the resource does not belong to the product
func IsLegal(res client.Object, productName string) error {
	if !res.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("resource %s is terminating", res.GetName())
	}

	if !IsBelongsToProduct(res, productName) {
		return fmt.Errorf("resource %s is not belongs to product", res.GetName())
	}
	return nil
}

// IsBelongsToProduct check resouces is maintain by nautes
func IsBelongsToProduct(res client.Object, productName string) bool {
	if res == nil {
		return false
	}

	labels := res.GetLabels()
	name, ok := labels[nautescrd.LABEL_FROM_PRODUCT]
	if !ok || name != productName {
		return false
	}
	return true
}
