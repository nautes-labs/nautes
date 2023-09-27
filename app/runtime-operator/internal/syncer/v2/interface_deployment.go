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

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewDeployment func(opt v1alpha1.Component, info *ComponentInitInfo) (Deployment, error)

type Deployment interface {
	Component

	Product
	AddProductUser(ctx context.Context, request PermissionRequest) error
	DeleteProductUser(ctx context.Context, request PermissionRequest) error

	CreateApp(ctx context.Context, app Application) error
	DeleteApp(ctx context.Context, app Application) error
	// SyncApp(ctx context.Context, apps []Application, cache interface{}) (interface{}, error)
}

type Application struct {
	Resource
	Git          *ApplicationGit
	Destinations []Space
}

type ApplicationGit struct {
	URL      string
	Revision string
	Path     string
	// CodeRepo will record the source, if url comes from code repo
	CodeRepo string
}
