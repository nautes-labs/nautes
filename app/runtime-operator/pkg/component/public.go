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

package component

import (
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/database"
	configs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	"k8s.io/client-go/rest"
)

// Component is a tool used by runtime operators to complete runtime deployment.
type Component interface {
	// CleanUp is used by components to refresh and clean data.
	// Runtime operator will call this method at the end of reconcile.
	// The error returned will not cause reconcile failure, it will only be output to the log.
	CleanUp() error
	// GetComponentMachineAccount returns the machine account used by the component, and the runtime operator will give the account the resource access it should have.
	// If the machine account is not needed, return nil.
	GetComponentMachineAccount() *MachineAccount
}

// ComponentInitInfo contains the public information for this reconcile.
// It is used for component initialization.
type ComponentInitInfo struct {
	// ClusterConnectionInfo records the access information of the cluster.
	ClusterConnectInfo ClusterConnectInfo
	// ClusterName is the name of the cluster where the runtime is located.
	ClusterName string
	// NautesResourceSnapshot store snapshots of resources related to runtime.
	NautesResourceSnapshot database.Snapshot
	// RuntimeName is the name of the runtime to be deployed this time.
	RuntimeName string
	// NautesConfig is nautes global config.
	NautesConfig configs.Config
	// Components is a reference to the component implementations in cluster.
	// Different cluster work type will implement different components.
	// The components that will be initialized in deployment cluster are:
	//   - Deployment
	//   - MultiTenant
	//   - SecretManagement
	//   - SecretSync
	//
	// The components that will be initialized in pipeline cluster are:
	//   - Deployment
	//   - MultiTenant
	//   - SecretManagement
	//   - SecretSync
	//   - EventListener
	//   - Pipeline
	//
	// Warning:
	//   The components are initialized in the following order:
	//   - SecretManagement
	//   - SecretSync
	//   - Deployment
	//   - MultiTenant
	//   - EventListener
	//   - Pipeline
	Components *ComponentList
	// PipelinePluginManager is an instance that manages pipeline plugins.
	PipelinePluginManager PipelinePluginManager
	// EventSourceSearchEngine is an instance that can search data in event source.
	EventSourceSearchEngine EventSourceSearchEngine
}

// ComponentList contains pointers to component implementations.
type ComponentList struct {
	// Deployment is a reference to deployment implementation.
	Deployment Deployment
	// MultiTenant is a reference to multiTenant implementation.
	MultiTenant MultiTenant
	// SecretManagement is a reference to secretManagement implementation.
	SecretManagement SecretManagement
	// SecretSync is a reference to secretSync implementation.
	SecretSync SecretSync
	// EventListener is a reference to eventListener implementation.
	EventListener EventListener
	// Pipeline is a reference to pipeline implementation.
	Pipeline Pipeline
}

// ClusterConnectInfo records the access method of the target cluster
type ClusterConnectInfo struct {
	// ClusterKind is the cluster kind in nautes resource 'Cluster'.
	ClusterKind v1alpha1.ClusterKind
	// Kubernetes record the access information of the kubernetes cluster.
	// It should be an administrator account.
	// If cluster kind is 'CLUSTER_KIND_KUBERNETES', it should not be nil.
	Kubernetes *ClusterConnectInfoKubernetes
}

// ClusterConnectInfoKubernetes store the attributes for kubernetes client initialization.
type ClusterConnectInfoKubernetes struct {
	// Config is a rest.Config witch used to initialize k8s client.
	Config *rest.Config
}

// ResourceMetaData records the metadata information of resource,
// Resource should inherit this structure.
type ResourceMetaData struct {
	// Product is the product name to which resource belongs.
	Product string `json:"product"`
	// Name is the name of resource.
	Name string `json:"name"`
}

// SpaceType is the type of space.
// It is determined by the type of cluster.
type SpaceType string

const (
	SpaceTypeKubernetes = "kubernetes" // Space in kubernetes
	SpaceTypeServer     = "server"     // Space in server
)

// Space is a logical workspace.
// Space should be completely isolated from each other.
type Space struct {
	ResourceMetaData
	// SpaceType is the type of space.
	SpaceType SpaceType `json:"spaceType"`
	// Kubernetes records kubernetes space info.
	Kubernetes *SpaceKubernetes `json:"kubernetes,omitempty"`
	// Server records space info in servers.
	Server *SpaceServer `json:"server,omitempty"`
}

// SpaceStatus records the metadata information and authorization of the space.
type SpaceStatus struct {
	Space
	// Accounts records the accounts who can use this space.
	Accounts []string
}

// SpaceKubernetes records the resources of space in k8s.
type SpaceKubernetes struct {
	// Namespace is the name of namespace in kubernetes.
	Namespace string
}

// SpaceServer records the resource usage of space on the host.
type SpaceServer struct {
	// Home 是 space 在主机中的绝对路径。
	Home string
}

// MachineAccount is a logical concept, it is the carrier of various resource permissions, e.g.:
//   - Permission to use spaces.
//   - Permission to access code repos.
//   - Permission to access artifact repos.
//
// MachineAccount does not necessarily belong to a product. It can also belongs to a component.
// Warning:
//
//		If account is used by component, it is important to avoid name conflicts.
//		There is no conflict prevention mechanism.
//
//	 Example:
//
//		  Component A use "sample" as name, and account has code repo c1's readonly permission,
//		  Component B use "sample" as name too, and account has code repo c2's readonly permission.
//		  Finally, account "sample" has both c1 and c2 readonly permission.
type MachineAccount struct {
	// Name is the name of the machine account.
	// It is unique in the cluster
	Name string
	// Product is the product name account belongs to.
	// Account can not belong to any product.
	Product string
	// Spaces records the list of spaces available to the account
	Spaces []string
}

// AuthType is the classification of authentication methods
type AuthType string

const (
	AuthTypeKubernetesServiceAccount AuthType = "ServiceAccount"
	AuthTypeUserPassword             AuthType = "Password"
	AuthTypeKeypair                  AuthType = "Keypair"
	AuthTypeToken                    AuthType = "Token"
)

// AuthInfo stores information related to identity authentication.
type AuthInfo struct {
	// OriginName is the name of the system providing authentication services.
	OriginName string
	// AccountName is the account name in the authentication system.
	AccountName string
	// AuthType is the authentication method used for authentication information.
	AuthType AuthType
	// ServiceAccounts stores a set of service account information that can be used to verify identity.
	ServiceAccounts []AuthInfoServiceAccount
}

// AuthInfoServiceAccount records service account information.
type AuthInfoServiceAccount struct {
	// ServiceAccount is the name of the service account in kubernetes.
	ServiceAccount string
	// Namespace is the namespace of service account.
	Namespace string
}

// RequestScope represents the scope of authorization.
type RequestScope string

const (
	RequestScopeProduct = "Product"
	RequestScopeProject = "Project"
	RequestScopeAccount = "Account"
)

// PermissionRequest is a definition of a permission request that contains information about:
// - The resources to be authorized.
// - The authorized objects.
// - The scope of permissions that have been requested.
type PermissionRequest struct {
	// Resource is the metadata information of the authorized resource.
	Resource ResourceMetaData
	// User is the object that requests permission.
	// If request scope is 'RequestScopeProduct', user is the 'ProductName' in nautes resource 'Product'.
	// If request scope is 'RequestScopeProject', user is the name of nautes resource 'Project'.
	// If request scope is 'RequestScopeAccount', user is the name of struct 'MachineAccount'.
	// Different authorization scopes with the same name are considered different objects.
	User         string
	RequestScope RequestScope
	// TODO: This is not implemented.
	// Permission is the permission required by the user.
	Permission Permission
}

type Permission struct {
	Admin      bool
	Permission int
}

// ComponentType is the definition of component classification
type ComponentType string

const (
	ComponentTypePipeline ComponentType = "pipeline"
)
