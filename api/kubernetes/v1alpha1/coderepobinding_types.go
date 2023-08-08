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

const (
	CodeRepoBindingPermissionReadOnly  = "readonly"
	CodeRepoBindingPermissionReadWrite = "readwrite"
)

// CodeRepoBindingSpec defines the desired state of CodeRepoBinding
type CodeRepoBindingSpec struct {
	// Authorized Code Repository.
	CodeRepo string `json:"codeRepo,omitempty"`
	// The Code repo is authorized to this product or projects under it.
	Product string `json:"product,omitempty"`
	// If the project list is empty, it means that the code repo is authorized to the product.
	// If the project list has values, it means that the code repo is authorized to the specified projects.
	// +optional
	Projects []string `json:"projects,omitempty"`
	// Authorization Permissions, readwrite or readonly.
	Permissions string `json:"permissions,omitempty"`
}

// CodeRepoBindingStatus defines the observed state of CodeRepoBinding
type CodeRepoBindingStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="PRODUCTRESOURCENAME",type=string,JSONPath=".spec.product"
//+kubebuilder:printcolumn:name="CODEREPORESOURCENAME",type=string,JSONPath=".spec.coderepo"

// CodeRepoBinding is the Schema for the coderepobindings API
type CodeRepoBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeRepoBindingSpec   `json:"spec,omitempty"`
	Status CodeRepoBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CodeRepoBindingList contains a list of CodeRepoBinding
type CodeRepoBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeRepoBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeRepoBinding{}, &CodeRepoBindingList{})
}
