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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArtifactRepoSpec defines the desired state of ArtifactRepo
type ArtifactRepoSpec struct {
	ArtifactRepoProvider string `json:"artifactRepoProvider" yaml:"artifactRepoProvider"`
	Product              string `json:"product" yaml:"product"`
	// +nullable
	// +optional
	Projects []string `json:"projects" yaml:"projects"`
	RepoName string   `json:"repoName" yaml:"repoName"`
	// remote, local, virtual...
	// +kubebuilder:validation:Enum=remote;local;virtual
	RepoType string `json:"repoType" yaml:"repoType"`
	// maven, python, go...
	// +kubebuilder:validation:Enum=maven;python;go
	PackageType string `json:"packageType" yaml:"packageType"`
}

// ArtifactRepoStatus defines the observed state of ArtifactRepo
type ArtifactRepoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ArtifactRepo is the Schema for the artifactrepoes API
type ArtifactRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactRepoSpec   `json:"spec,omitempty"`
	Status ArtifactRepoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArtifactRepoList contains a list of ArtifactRepo
type ArtifactRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArtifactRepo{}, &ArtifactRepoList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (a *ArtifactRepo) Compare(obj client.Object) (bool, error) {
	return false, nil
}
