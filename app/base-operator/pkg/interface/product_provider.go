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

package syncer

import (
	"context"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type ProductProviderSyncer interface {
	// Sync create|update|remove product resource by product provider
	Sync(context.Context, nautescrd.ProductProvider) error
}

// ProductProvider is an object used to obtain product information from the product database
type ProductProvider interface {
	GetProducts() ([]nautescrd.Product, error)
	GetProductMeta(ctx context.Context, ID string) (ProductMeta, error)
	GetCodeRepoProvider(ctx context.Context) (CodeRepoProvider, error)
}

// ProductMeta record product meta data
type ProductMeta struct {
	// Product ID
	ID string
	// The ID of the git code repo that records product information
	MetaID string
}

type CodeRepoProvider struct {
	// Code repo provider name in tenant k8s
	Name string
}
