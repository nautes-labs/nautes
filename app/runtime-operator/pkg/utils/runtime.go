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
	"context"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	interfaces "github.com/nautes-labs/nautes/app/runtime-operator/pkg/interface"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetClusterByRuntime(ctx context.Context, k8sClient client.Client, nautesNamespace string, runtime interfaces.Runtime) (*nautescrd.Cluster, error) {
	env := &nautescrd.Environment{}
	key := types.NamespacedName{
		Namespace: runtime.GetProduct(),
		Name:      runtime.GetDestination(),
	}
	if err := k8sClient.Get(ctx, key, env); err != nil {
		return nil, err
	}

	cluster := &nautescrd.Cluster{}
	key = types.NamespacedName{
		Namespace: nautesNamespace,
		Name:      env.Spec.Cluster,
	}

	if err := k8sClient.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

func GetReservedNamespacesByProduct(cluster nautescrd.Cluster, productID string) []string {
	productName := GetProductNameByIDInCluster(cluster, productID)
	if productName == "" {
		return nil
	}

	reservedNamespaces := []string{}
	for namespace, products := range cluster.Spec.ReservedNamespacesAllowedProducts {
		for _, name := range products {
			if name == productName {
				reservedNamespaces = append(reservedNamespaces, namespace)
				break
			}
		}
	}

	return reservedNamespaces
}

func GetProductNameByIDInCluster(cluster nautescrd.Cluster, productID string) string {
	productName := ""
	for name, id := range cluster.Status.ProductIDMap {
		if id == productID {
			productName = name
			break
		}
	}
	return productName
}
