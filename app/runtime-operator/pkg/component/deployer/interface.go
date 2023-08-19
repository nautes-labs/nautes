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
