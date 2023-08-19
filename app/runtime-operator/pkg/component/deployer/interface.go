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

package deployer

import (
	"context"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component/initinfo"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/datasource"
)

type NewDeployer func(namespace string, task initinfo.ComponentInitInfo, db datasource.DataSource) (Deployer, error)

type Deployer interface {
	// GetNewResource() {resources}
	DeployApp(ctx context.Context, app Application) error
	UnDeployApp(ctx context.Context, app Application) error
	// GrantPermission()
	// RevokePermission()
}

type Application struct {
	Name        string
	Git         *ApplicationGit
	Destination Destination
}

type ApplicationGit struct {
	URL      string
	Revision string
	Path     string
}

type Destination struct {
	Kubernetes *DestinationKubernetes
}

type DestinationKubernetes struct {
	Namespace string
}
