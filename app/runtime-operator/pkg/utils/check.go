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

package utils

import (
	"fmt"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// IsLegal used to check resources is availabel for reconcile.
// It will return two vars, pass or not and the reason if check is failed.
func IsLegal(res client.Object, productName string) (string, bool) {
	if err := CheckResourceOperability(res, productName); err != nil {
		return err.Error(), false
	}
	return "", true
}

func CheckResourceOperability(res client.Object, productName string) error {
	if !res.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("resouce %s is terminating", res.GetName())
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
	name, ok := labels[nautescrd.LABEL_BELONG_TO_PRODUCT]
	if !ok || name != productName {
		return false
	}
	return true
}

// IsOwner check k8s resource is belongs to another resource
func IsOwner(owner, object client.Object, scheme *runtime.Scheme) bool {
	if owner == nil {
		return false
	}

	if owner.GetNamespace() != object.GetNamespace() || object.GetNamespace() == "" {
		return false
	}

	gvk, err := apiutil.GVKForObject(owner, scheme)
	if err != nil {
		return false
	}

	ownerInfos := object.GetOwnerReferences()
	for _, ownerInfo := range ownerInfos {
		if ownerInfo.APIVersion == gvk.GroupVersion().String() &&
			ownerInfo.Kind == gvk.Kind &&
			ownerInfo.UID == owner.GetUID() &&
			ownerInfo.Name == owner.GetName() {
			return true
		}
	}
	return false
}
