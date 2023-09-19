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
	"k8s.io/apimachinery/pkg/runtime"
)

type ManifestSource struct {
	CodeRepo       string `json:"codeRepo,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Path           string `json:"path"`
}

// DeploymentRuntimeSpec defines the desired state of DeploymentRuntime
type DeploymentRuntimeSpec struct {
	Product string `json:"product,omitempty" yaml:"product"`
	// +optional
	ProjectsRef    []string                      `json:"projectsRef,omitempty" yaml:"projectsRef"`
	ManifestSource ManifestSource                `json:"manifestSource,omitempty" yaml:"manifestSource"`
	Destination    DeploymentRuntimesDestination `json:"destination" yaml:"destination"`
}

type DeploymentRuntimesDestination struct {
	Environment string `json:"environment"`
	// +optional
	Namespaces []string `json:"namespaces"`
}

// DeploymentRuntimeStatus defines the observed state of DeploymentRuntime
type DeploymentRuntimeStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions"`
	// +optional
	Cluster string `json:"cluster,omitempty"`
	// +optional
	// +nullable
	IllegalProjectRefs []IllegalProjectRef `json:"illegalProjectRefs,omitempty"`
	// +nullable
	DeployHistory *DeployHistory `json:"deployHistory,omitempty" yaml:"deployHistory"`
	// +optional
	// +nullable
	DeployStatus *runtime.RawExtension `json:"deployStatus,omitempty" yaml:"deployStatus"`
}

type IllegalProjectRef struct {
	ProjectName string `json:"projectName"`
	Reason      string `json:"reason"`
}

type DeployHistory struct {
	ManifestSource ManifestSource                `json:"manifestSource"`
	Destination    DeploymentRuntimesDestination `json:"destination"`
	// The URL of the coderepo in the last successfully deployed runtime
	Source string `json:"source"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=dr
//+kubebuilder:printcolumn:name="Destination",type=string,JSONPath=".spec.destination.environment"
//+kubebuilder:printcolumn:name="CodeRepo",type=string,JSONPath=".spec.manifestSource.codeRepo"

// DeploymentRuntime is the Schema for the deploymentruntimes API
type DeploymentRuntime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentRuntimeSpec   `json:"spec,omitempty"`
	Status DeploymentRuntimeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentRuntimeList contains a list of DeploymentRuntime
type DeploymentRuntimeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentRuntime `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentRuntime{}, &DeploymentRuntimeList{})
}
