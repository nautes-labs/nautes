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

// CodeRepoProviderSpec defines the desired state of CodeRepoProvider
type CodeRepoProviderSpec struct {
	HttpAddress string `json:"httpAddress"`
	SSHAddress  string `json:"sshAddress"`
	// +kubebuilder:validation:Type=string
	ApiServer string `json:"apiServer"`
	// +kubebuilder:validation:Enum=gitlab;github
	ProviderType string `json:"providerType"`
}

// CodeRepoProviderStatus defines the observed state of CodeRepoProvider
type CodeRepoProviderStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CodeRepoProvider is the Schema for the coderepoproviders API
type CodeRepoProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeRepoProviderSpec   `json:"spec,omitempty"`
	Status CodeRepoProviderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CodeRepoProviderList contains a list of CodeRepoProvider
type CodeRepoProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeRepoProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeRepoProvider{}, &CodeRepoProviderList{})
}

func (c *CodeRepoProvider) Compare(obj client.Object) (bool, error) {
	return false, nil
}
