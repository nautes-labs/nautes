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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterType string
type ClusterKind string
type ClusterUsage string
type ClusterSyncStatus string
type ClusterWorkType string

const (
	CLUSTER_KIND_KUBERNETES ClusterKind = "kubernetes"
)

const (
	CLUSTER_TYPE_PHYSICAL ClusterType = "physical"
	CLUSTER_TYPE_VIRTUAL  ClusterType = "virtual"
)

const (
	CLUSTER_USAGE_HOST   ClusterUsage = "host"
	CLUSTER_USAGE_WORKER ClusterUsage = "worker"
)

const (
	ClusterWorkTypeDeployment ClusterWorkType = "deployment"
	ClusterWorkTypePipeline   ClusterWorkType = "pipeline"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// +kubebuilder:validation:Pattern=`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&\/\/=]*)`
	ApiServer string `json:"apiServer" yaml:"apiServer"`
	// +kubebuilder:validation:Enum=physical;virtual
	ClusterType ClusterType `json:"clusterType" yaml:"clusterType"`
	// +optional
	// +kubebuilder:default:=kubernetes
	// +kubebuilder:validation:Enum=kubernetes
	ClusterKind ClusterKind `json:"clusterKind" yaml:"clusterKind"`
	// the usage of cluster, for user use it directry or deploy vcluster on it
	// +kubebuilder:validation:Enum=host;worker
	Usage ClusterUsage `json:"usage" yaml:"usage"`
	// +optional
	HostCluster string `json:"hostCluster,omitempty" yaml:"hostCluster"`
	// +optional
	// PrimaryDomain is used to build the domain of components within the cluster
	PrimaryDomain string `json:"primaryDomain,omitempty" yaml:"primaryDomain"`
	// +optional
	// +kubebuilder:validation:Enum="";pipeline;deployment
	// pipeline or deployment, when the cluster usage is 'worker', the WorkType is required.
	WorkerType     ClusterWorkType `json:"workerType,omitempty" yaml:"workerType"`
	ComponentsList ComponentsList  `json:"componentsList"`
	// +optional
	// ReservedNamespacesAllowedProducts key is namespace name, value is the product name list witch can use namespace.
	ReservedNamespacesAllowedProducts map[string][]string `json:"reservedNamespacesAllowedProducts"`
	// +optional
	// ReservedNamespacesAllowedProducts key is product name, value is the list of cluster resources.
	ProductAllowedClusterResources map[string][]ClusterResourceInfo `json:"productAllowedClusterResources"`
}

type ClusterResourceInfo struct {
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`
	// +kubebuilder:validation:MinLength=1
	Group string `json:"group"`
}

type Component struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

// ComponentsList declares the specific components used by the cluster
type ComponentsList struct {
	// +optional
	CertMgt *Component `json:"certMgt"`
	// +optional
	Deployment *Component `json:"deployment"`
	// +optional
	EventListener *Component `json:"eventListener"`
	// +optional
	IngressController *Component `json:"ingressController"`
	// +optional
	MultiTenant *Component `json:"multiTenant"`
	// +optional
	Pipeline *Component `json:"pipeline"`
	// +optional
	ProgressiveDelivery *Component `json:"progressiveDelivery"`
	// +optional
	SecretMgt *Component `json:"secretMgt"`
	// +optional
	SecretSync *Component `json:"secretSync"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	MgtAuthStatus *MgtClusterAuthStatus `json:"mgtAuthStatus,omitempty" yaml:"mgtAuthStatus"`
	// +optional
	Sync2ArgoStatus *SyncCluster2ArgoStatus `json:"sync2ArgoStatus,omitempty" yaml:"sync2ArgoStatus"`
	// +optional
	EntryPoints map[string]ClusterEntryPoint `json:"entryPoints,omitempty" yaml:"entryPoints"`
	// +optional
	Warnings []Warning `json:"warnings"`
	// +optional
	// PruoductIDMap records the corresponding relationship between product name and product in kubernetes.
	ProductIDMap map[string]string `json:"productIDMap"`
}

type ServiceType string

const (
	ServiceTypeNodePort     ServiceType = "NodePort"
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
	ServiceTypeExternalName ServiceType = "ExternalName"
)

type ClusterEntryPoint struct {
	// The port of entry point service
	HTTPPort  int32       `json:"httpPort,omitempty" yaml:"httpPort"`
	HTTPSPort int32       `json:"httpsPort,omitempty" yaml:"httpsPort"`
	Type      ServiceType `json:"type,omitempty" yaml:"type"`
}

type MgtClusterAuthStatus struct {
	LastSuccessSpec string      `json:"lastSuccessSpec" yaml:"lastSuccessSpec"`
	LastSuccessTime metav1.Time `json:"lastSuccessTime" yaml:"lastSuccessTime"`
	SecretID        string      `json:"secretID" yaml:"secretID"`
}

type SyncCluster2ArgoStatus struct {
	LastSuccessSpec string      `json:"lastSuccessSpec" yaml:"lastSuccessSpec"`
	LastSuccessTime metav1.Time `json:"lastSuccessTime" yaml:"lastSuccessTime"`
	SecretID        string      `json:"secretID" yaml:"secretID"`
}

type Warning struct {
	Type string `json:"type"`
	// From is which operator recorded this warning.
	From string `json:"from"`
	// ID records the unique identifier of the warning.
	ID      string `json:"id"`
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="API SERVER",type=string,JSONPath=`.spec.apiserver`
//+kubebuilder:printcolumn:name="USAGE",type=string,JSONPath=".spec.usage"
//+kubebuilder:printcolumn:name="HOST",type=string,JSONPath=".spec.hostcluster"
//+kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
