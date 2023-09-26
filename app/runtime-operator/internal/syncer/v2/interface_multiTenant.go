package syncer

import (
	"context"
	"errors"
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewMultiTenant func(opt v1alpha1.Component, info *ComponentInitInfo) (MultiTenant, error)

type MultiTenant interface {
	Component
	Product

	// Space use to manager a logical space in environment.

	CreateSpace(ctx context.Context, productName string, name string) error
	DeleteSpace(ctx context.Context, productName string, name string) error
	GetSpace(ctx context.Context, productName, name string) (*SpaceStatus, error)
	ListSpaces(ctx context.Context, productName string, opts ...ListOption) ([]SpaceStatus, error)
	AddSpaceUser(ctx context.Context, request PermissionRequest) error
	DeleteSpaceUser(ctx context.Context, request PermissionRequest) error

	CreateUser(ctx context.Context, productName, name string) error
	DeleteUser(ctx context.Context, productName, name string) error
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

type ProductError struct {
	error
	reason string
}

type ErrorReason string

const (
	ErrorReasonUserNotFound = "UserNotFound"
)

func UserNotFound(err error, name string) ProductError {
	return ProductError{
		error:  fmt.Errorf("user %s not found: %w", name, err),
		reason: ErrorReasonUserNotFound,
	}
}

func IsUserNotFound(err error) bool {
	var productErr ProductError
	if errors.As(err, &productErr) {
		if productErr.reason == ErrorReasonUserNotFound {
			return true
		}
	}
	return false
}
