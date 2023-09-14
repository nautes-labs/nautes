package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewDeployment func(opt v1alpha1.Component, info ComponentInitInfo) (Deployment, error)

type Deployment interface {
	Component

	// The cache will be stored and passed based on the product name.
	Product
	AddProductUser(ctx context.Context, request PermissionRequest) error
	DeleteUProductUser(ctx context.Context, request PermissionRequest) error

	// SyncApp should deploy the given apps, and clean up expired apps in cache.
	// All apps share one cache.
	SyncApp(ctx context.Context, apps []Application, cache interface{}) (interface{}, error)
	SyncAppUsers(ctx context.Context, requests []PermissionRequest, cache interface{}) (interface{}, error)
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
