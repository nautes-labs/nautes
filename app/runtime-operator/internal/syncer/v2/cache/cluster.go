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

package cache

import (
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"gopkg.in/yaml.v2"
)

type ClusterUsage struct {
	Products map[string]ProductUsage `yaml:"products"`
}

// GetAccountOwner will return the product id who is using account name, if account name is not used, it will return empty
func (cu *ClusterUsage) GetAccountOwner(name string) string {
	var owner string
	for productID, product := range cu.Products {
		for accountName := range product.Account.Accounts {
			if accountName == name {
				owner = productID
				break
			}
		}
		if owner != "" {
			break
		}
	}
	return owner
}

func (cu *ClusterUsage) AddRuntimeUsage(runtime v1alpha1.Runtime) {
	productID := runtime.GetProduct()

	_, ok := cu.Products[productID]
	if !ok {
		cu.Products[productID] = ProductUsage{
			Runtimes: utils.NewStringSet(),
		}
	}

	cu.Products[productID].Runtimes.Insert(runtime.GetName())
}

func (cu *ClusterUsage) DeleteRuntimeUsage(runtime v1alpha1.Runtime) {
	productID := runtime.GetProduct()
	_, ok := cu.Products[productID]
	if !ok {
		return
	}

	cu.Products[productID].Runtimes.Delete(runtime.GetName())
}

type ProductUsage struct {
	Runtimes utils.StringSet `json:"runtimes"`
	Space    SpaceUsage      `json:"space"`
	Account  AccountUsage    `json:"account"`
}

func NewClustersUsage(usage string) (*ClusterUsage, error) {
	clusterUsage := &ClusterUsage{
		Products: map[string]ProductUsage{},
	}
	if err := yaml.Unmarshal([]byte(usage), clusterUsage); err != nil {
		return nil, err
	}
	return clusterUsage, nil
}

func NewEmptyProductUsage() ProductUsage {
	return ProductUsage{
		Runtimes: utils.NewStringSet(),
		Account: AccountUsage{
			Accounts: map[string]AccountResource{},
		},
		Space: SpaceUsage{},
	}
}
