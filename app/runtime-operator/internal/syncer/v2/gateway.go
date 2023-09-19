package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type NewGateway func(opt v1alpha1.Component, info *ComponentInitInfo) (Gateway, error)

type Gateway interface {
	Component

	// CreateEntryPoint will create an entrypoint on gateway.
	// EntryPoint may forward to a kubernetes service or a remote Service.
	// Gateway should support at least one of them.
	CreateEntryPoint(ctx context.Context, entrypoint EntryPoint) error
	RemoveEntryPoint(ctx context.Context, entrypoint EntryPoint) error
}

type EntryPoint struct {
	Name string
	// Source is the entrypoint the user requested. It contained protocol, domain, subpath, and port.
	Source EntryPointSource
	// Destination is where the request should forward to. It may be a kubernetes service or remote service.
	Destination EntryPointDestination
}

type EntryPointSource struct {
	Domain   string
	Path     string
	Port     int
	Protocol string
}

type DestinationType string

const (
	DestinationTypeKubernetes    = "kubernetes"
	DestinationTypeRemoteService = "remoteService"
)

type EntryPointDestination struct {
	Type              DestinationType
	KubernetesService *KubernetesService
	RemoteService     *RemoteService
}

type RemoteService struct {
	Domain   string
	Port     int
	Protocol string
	Path     string
}

type KubernetesService struct {
	Name      string
	Namespace string
	Port      int
}
