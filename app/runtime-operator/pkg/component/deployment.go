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

// NewDeployment will return a deployment implementation.
// opt is the user defined component options.
// info is the information can help deployment to finish jobs.
type NewDeployment func(opt v1alpha1.Component, info *ComponentInitInfo) (Deployment, error)

// Deployment is a tool for runtime operator to manage resource deployment.
// Its main functions are two:
// - Manage the lifecycle of resources in the cluster.
// - Managing user's operation permissions on resources.
//
// It has two concepts:
//   - Product
//     Products in Deployment are equivalent to those in Nautes. It records the application information of products in the cluster, as well as the user's operation permissions on resources.
//   - Application
//     An application represents a set of resources to be deployed, and the application must belong to a product.
type Deployment interface {
	Component

	// CreateProduct is to create a product in the cluster.
	//
	// Return:
	// - If the product already exists, return nil directly.
	CreateProduct(ctx context.Context, name string) error

	// DeleteProduct will delete the product, its applications, and the authorization information related to the product in the cluster.
	//
	// Return:
	// - If the product does not exist, use the 'ProductNotFound' method to return an error.
	DeleteProduct(ctx context.Context, name string) error

	// AddProductUser authorizes the user to manage products.
	//
	// Additional notes:
	// The name in request is the product name to be authorized.
	//
	// Return:
	// - If the name of the request does not exist, use the method 'ProductNotFound' to return an error.
	AddProductUser(ctx context.Context, request PermissionRequest) error

	// DeleteProductUser revokes the user's management authority on the product.
	//
	// Additional notes:
	// The name in request is the product name to be revoked.
	//
	// Return:
	// - If the product does not exist, return nil.
	DeleteProductUser(ctx context.Context, request PermissionRequest) error

	// CreateApp will deploy resources in the cluster based on the data sources in the app.
	//
	// Additional notes:
	// - When deploying resources, only resources within the app's destinations scope are deployed. Resources outside the scope are not considered an error.
	// - When the app has been deployed (the same product with the same name is considered to be the same app), deploy resources based on the new app and clean up invalid resources.
	//
	// Return:
	// - If no available data source can be found in the app, an error is returned.
	// - If the product name in the app is empty, return an error.
	// - If the product name does not exist in the app, use the method "ProductNotFound" to return an error.
	CreateApp(ctx context.Context, app Application) error

	// DeleteApp will delete the specified application based on the product and name in the app.
	//
	// Return:
	// - If the product does not exist, return nil.
	DeleteApp(ctx context.Context, app Application) error
}

// The Application declares a set of resources to be deployed and the deployment restrictions.
type Application struct {
	ResourceMetaData
	// Git holds git specific options.
	// If the deployment files come from code repo, it should not be nil.
	Git *ApplicationGit
	// Destinations is the spaces that will be used for deployment.
	Destinations []Space
}

// ApplicationGit contains all required information about the source of an application.
type ApplicationGit struct {
	// URL is the URL of code repo.
	URL string
	// Revision is the target revision in code repo.
	Revision string
	// Path is the location where the deployment files are stored.
	Path string
	// CodeRepo is the name of the nautes resource 'CodeRepo'.
	// If URL does not come from the resource 'CodeRepo', it should be empty.
	CodeRepo string
}
