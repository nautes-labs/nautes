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

// ArtifactRepoProviderSpec defines the desired state of ArtifactRepoProvider
type ArtifactRepoProviderSpec struct {
	URL          string `json:"url" yaml:"url"`
	APIServer    string `json:"apiServer" yaml:"apiServer"`
	ProviderType string `json:"providerType" yaml:"providerType"`
}

// ArtifactRepoProviderStatus defines the observed state of ArtifactRepoProvider
type ArtifactRepoProviderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ArtifactRepoProvider is the Schema for the artifactrepoproviders API
type ArtifactRepoProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactRepoProviderSpec   `json:"spec,omitempty"`
	Status ArtifactRepoProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArtifactRepoProviderList contains a list of ArtifactRepoProvider
type ArtifactRepoProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArtifactRepoProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArtifactRepoProvider{}, &ArtifactRepoProviderList{})
}

func (a *ArtifactRepoProvider) Compare(obj client.Object) (bool, error) {
	return false, nil
}
