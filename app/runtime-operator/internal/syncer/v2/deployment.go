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
