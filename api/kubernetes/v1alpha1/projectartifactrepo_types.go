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

// ProjectArtifactRepoSpec defines the desired state of ProjectArtifactRepo
type ProjectArtifactRepoSpec struct {
	Project      string `json:"project"`
	ArtifactRepo string `json:"artifactRepo"`
}

// ProjectArtifactRepoStatus defines the observed state of ProjectArtifactRepo
type ProjectArtifactRepoStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectArtifactRepo is the Schema for the projectartifactrepoes API
type ProjectArtifactRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectArtifactRepoSpec   `json:"spec,omitempty"`
	Status ProjectArtifactRepoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectArtifactRepoList contains a list of ProjectArtifactRepo
type ProjectArtifactRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectArtifactRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectArtifactRepo{}, &ProjectArtifactRepoList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (p *ProjectArtifactRepo) Compare(obj client.Object) (bool, error) {
	return false, nil
}
