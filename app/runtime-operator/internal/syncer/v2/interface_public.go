package syncer

import (
	"context"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/client-go/rest"
)

// Component is the interface that all components should implement.
// These mehtods will called by syncer
type Component interface {
	// When the component generates cache information, implement this method to clean datas.
	// This method will be automatically called by the syncer after each tuning is completed.
	CleanUp() error
}

// Product is a universal interface definition for project management.
type Product interface {
	CreateProduct(ctx context.Context, name string) error
	DeleteProduct(ctx context.Context, name string) error
	GetProduct(ctx context.Context, name string) (*ProductStatus, error)
}

type ProductStatus struct {
	Name     string
	Projects []string
	Users    []string
}

// ComponentInitInfo is used to assist the component in initializing itself.
type ComponentInitInfo struct {
	ClusterConnectInfo ClusterConnectInfo
	ClusterName        string
	RuntimeType        v1alpha1.RuntimeType
	// Snapshot of nautes resources in runtime's product
	NautesDB     database.Database
	NautesConfig configs.Config
	Components   *ComponentList
}

type ComponentList struct {
	Deployment       Deployment
	MultiTenant      MultiTenant
	SecretManagement SecretManagement
	SecretSync       SecretSync
	EventListener    EventListener
}

type ClusterConnectInfo struct {
	Type       v1alpha1.ClusterKind
	Kubernetes *ClusterConnectInfoKubernetes
}

type ClusterConnectInfoKubernetes struct {
	Config *rest.Config
}

type Resource struct {
	Product string
	Name    string
}

type SpaceType string

const (
	SpaceTypeKubernetes = "kubernetes"
	SpaceTypeHost       = "host"
)

type Space struct {
	Resource
	SpaceType  SpaceType
	Kubernetes SpaceKubernetes
}

type SpaceStatus struct {
	Space
	Users []string
}

type SpaceKubernetes struct {
	Namespace string
}

type UserType string

const (
	UserTypeHuman   = "human"
	UserTypeMachine = "machine"
)

type User struct {
	Resource
	Role     []string
	UserType UserType
	AuthInfo *Auth
}

type Auth struct {
	Kubernetes []AuthKubernetes
}

type AuthKubernetes struct {
	ServiceAccount string
	Namespace      string
}

type RequestScope string

const (
	RequestScopeProduct = "Product"
	ReqeustScopProject  = "Project"
	RequestScopeUser    = "User"
)

type PermissionRequest struct {
	RequestScope RequestScope
	Resource     Resource
	User         string
	Permission   Permission
}

type Permission struct {
	Admin      bool
	Permission int
}
