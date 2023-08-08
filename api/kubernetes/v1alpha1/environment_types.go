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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	Product string `json:"product"`
	Cluster string `json:"cluster"`
	EnvType string `json:"envType"`
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="PRODUCTRESOURCENAME",type=string,JSONPath=".spec.product"
//+kubebuilder:printcolumn:name="CLUSTER",type=string,JSONPath=".spec.cluster"
//+kubebuilder:printcolumn:name="ENVTYPE",type=string,JSONPath=".spec.envType"

// Environment is the Schema for the environments API
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (e *Environment) Compare(obj client.Object) (bool, error) {
	val, ok := obj.(*Environment)
	if !ok {
		return false, fmt.Errorf("the resource %s type is inconsistent", obj.GetName())
	}

	if val.Spec.Cluster == e.Spec.Cluster {
		return true, nil
	}

	return false, nil
}
