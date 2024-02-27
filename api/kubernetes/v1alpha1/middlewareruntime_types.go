/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	MiddlewareTypeRedis = "redis"
	MiddlewareTypeMongo = "mongo"
)

// MiddlewareRuntimeSpec defines the desired state of MiddlewareRuntime
type MiddlewareRuntimeSpec struct {
	// Destination is the target environment where the middleware should be deployed.
	Destination MiddlewareRuntimeDestination `json:"destination"`
	// +optional
	// AccessInfoName is the account used for deploying the middleware runtime.
	AccessInfoName string `json:"accessInfoName,omitempty"`
	// +optional
	// Account is the account used for running the middleware.
	Account string `json:"account,omitempty"`
	// +optional
	// +nullable
	// Caches is a list of cache middlewares.
	Caches []Middleware `json:"caches,omitempty"`
	// +optional
	// +nullable
	// DataBases is a list of database middlewares.
	DataBases []Middleware `json:"databases,omitempty"`
}

// MiddlewareRuntimeDestination represents the target environment where the middleware runtime should be deployed.
type MiddlewareRuntimeDestination struct {
	// +kubebuilder:validation:MinLength=1
	// Environment is the name of the environment where the runtime should be deployed.
	Environment string `json:"environment"`
	// +optional
	// Space is the default deployment location for the middleware. If this value is empty, the runtime name will be used as the value.
	Space string `json:"space,omitempty"`
}

// Middleware defines the middleware to be deployed.
type Middleware struct {
	// +kubebuilder:validation:MinLength=1
	// Name is the name of the deployed middleware. It is used to differentiate it from other middleware and must be unique within the runtime.
	Name string `json:"name"`
	// +optional
	// Space is the deployment location of the middleware. If this value is empty, the middleware will be deployed in the default location.
	Space string `json:"space,omitempty"`
	// +kubebuilder:validation:MinLength=1
	// Type is the type of the middleware.
	Type string `json:"type"`
	// Implementation is the implementation of the middleware.
	// If the middleware has multiple implementation modes, you can specify which mode to use by setting this field.
	// +optional
	Implementation string `json:"implementation,omitempty"`
	// +optional
	// +nullable
	// Labels are the labels to be added to the middleware.
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	// +nullable
	// Entrypoint represents the external entrypoint configuration for the middleware.
	Entrypoint *MiddlewareEntrypoint `json:"entrypoint,omitempty"`
	// +optional
	// +nullable
	// Redis represents the deployment information for middleware of type "redis".
	Redis *MiddlewareDeploymentRedis `json:"redis,omitempty"`
	// +optional
	// +nullable
	// CommonMiddlewareInfo represents the deployment information for middleware of common types.
	// For more information, please refer to the documentation at https://github.com/nautes-labs/nautes/docs/How_to_add_a_custom_middleware.md
	CommonMiddlewareInfo map[string]string `json:"commonMiddlewareInfo,omitempty"`
	// +optional
	// +nullable
	// InitAccessInfo represents the access information for the middleware.
	// If it is not empty, the middleware will be deployed with the access information.
	// If it is empty, the runtime operator will generate the access information for the middleware.
	// User should change the access information after the middleware is deployed.
	InitAccessInfo *MiddlewareInitAccessInfo `json:"initAccessInfo,omitempty"`
}

const (
	MiddlewareAccessInfoTypeNotSpecified = "NotSpecified"
	MiddlewareAccessInfoTypeUserPassword = "UserPassword"
)

// MiddlewareInitAccessInfo represents the access information for the middleware.
type MiddlewareInitAccessInfo struct {
	// +optional
	// +nullable
	UserPassword *MiddlewareInitAccessInfoUserPassword `json:"userPassword,omitempty"`
}

type MiddlewareInitAccessInfoUserPassword struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MiddlewareEntrypoint represents the configuration for a middleware entrypoint.
type MiddlewareEntrypoint struct {
	Domain   string `json:"domain,omitempty"`   // Domain represents the domain name for the entrypoint.
	Path     string `json:"path,omitempty"`     // Path represents the path for the entrypoint.
	Port     int    `json:"port,omitempty"`     // Port represents the port number for the entrypoint.
	Protocol string `json:"protocol,omitempty"` // Protocol represents the protocol for the entrypoint.
}

type MiddlewareStorage struct {
	// Size represents the size of the storage.
	Size string `json:"size,omitempty"`
}

// MiddlewareDeploymentRedis represents the configuration for deploying a Redis middleware.
type MiddlewareDeploymentRedis struct {
	// DeployType specifies the type of deployment for the middleware.
	DeployType string `json:"deployType,omitempty"`
	// NodeNumber specifies the number of nodes for the middleware deployment.
	NodeNumber int `json:"nodeNumber,omitempty"`
	// Version specifies the version of the middleware.
	Version string `json:"version,omitempty"`
	// Storage specifies the storage configuration for the middleware.
	Storage MiddlewareStorage `json:"storage,omitempty"`
	// OtherSettings contains additional settings for the middleware deployment.
	OtherSettings map[string]string `json:"otherSettings,omitempty"`
}

// MiddlewareRuntimeStatus defines the observed state of MiddlewareRuntime
type MiddlewareRuntimeStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	// Environment is the information of the environment where the runtime is deployed.
	Environment Environment `json:"environment,omitempty"`
	// Cluster represents the cluster information for deploying the runtime.
	// +optional
	// +nullable
	Cluster *Cluster `json:"cluster,omitempty"`
	// AccessInfoName is used during middleware deployment.
	// +optional
	AccessInfoName string `json:"accessInfoName,omitempty"`
	// Spaces represents the space usage of the middleware.
	// +optional
	// +nullable
	Spaces []string `json:"spaces,omitempty"`
	// EntryPoints represents the external entrypoint configuration for the middlewares in the runtime.
	// +optional
	// +nullable
	EntryPoints []MiddlewareEntrypoint `json:"entryPoints,omitempty"`
	// Middlewares represents the deployment status of the middleware.
	// +optional
	// +nullable
	Middlewares []MiddlewareStatus `json:"middlewares,omitempty"`
}

// MiddlewareStatus defines the status of a middleware.
type MiddlewareStatus struct {
	// Middleware stores the declaration of the middleware used in the last deployment.
	Middleware Middleware `json:"middleware"`
	// SecretID records the ID for retrieving middleware secret information.
	SecretID string `json:"secretID,omitempty"`
	// +optional
	// +nullable
	// Status represents the deployment status of the middleware. Its format is determined by the caller implementing the deployment.
	Status *runtime.RawExtension `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=mr

// MiddlewareRuntime is the Schema for the middlewareruntimes API
type MiddlewareRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MiddlewareRuntimeSpec   `json:"spec,omitempty"`
	Status MiddlewareRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MiddlewareRuntimeList contains a list of MiddlewareRuntime
type MiddlewareRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MiddlewareRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MiddlewareRuntime{}, &MiddlewareRuntimeList{})
}
