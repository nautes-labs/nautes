package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewMultiTenant func(opt v1alpha1.Component, info ComponentInitInfo) (MultiTenant, error)

type MultiTenant interface {
	Component
	Product

	// Space use to manager a logical space in environment.
	// It can be a namespace in kubernetes, a folder in host or some else.
	// The cache will be stored and passed based on the product name + space name.

	CreateSpace(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error)
	DeleteSpace(ctx context.Context, productName string, name string, cache interface{}) (interface{}, error)
	GetSpace(ctx context.Context, productName, name string) (*SpaceStatus, error)
	ListSpaces(ctx context.Context, productName string, opts ...ListOption) ([]SpaceStatus, error)
	AddSpaceUser(ctx context.Context, request PermissionRequest) error
	DeleteSpaceUser(ctx context.Context, request PermissionRequest) error

	// The cache will be stored and passed based on the product name + user name.

	CreateUser(ctx context.Context, productName, name string, cache interface{}) (interface{}, error)
	DeleteUser(ctx context.Context, productName, name string, cache interface{}) (interface{}, error)
	GetUser(ctx context.Context, productName, name string) (*User, error)
}

type ListOption func(*ListOptions)

type ListOptions struct {
	User                 string
	IgnoreDataInDeletion bool
}

func ByUser(name string) ListOption {
	return func(lo *ListOptions) { lo.User = name }
}
func IgnoreResourceInDeletion() ListOption {
	return func(lo *ListOptions) { lo.IgnoreDataInDeletion = true }
}
