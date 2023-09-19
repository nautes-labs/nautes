package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewSecretSync func(opt v1alpha1.Component, info *ComponentInitInfo) (SecretSync, error)

type SecretSync interface {
	Component

	// CreateSecret will create a secret object (sercret in kubernetes, file in host) from secret database to dest environment.
	CreateSecret(ctx context.Context, secretReq SecretRequest) error
	RemoveSecret(ctx context.Context, secretReq SecretRequest) error
}

type SecretRequest struct {
	Name        string
	Source      SecretInfo
	AuthInfo    Auth
	Destination SecretRequestDestination
}

type SecretRequestDestination struct {
	Name   string
	Space  Space
	Format string
}
