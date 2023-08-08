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
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LABEL_TENANT_MANAGEMENT = "coderepo.resource.nautes.io/tenant-management"
)

type Webhook struct {
	Events []string `json:"events" yaml:"events"`
}

// CodeRepoSpec defines the desired state of CodeRepo
type CodeRepoSpec struct {
	// +optional
	CodeRepoProvider string `json:"codeRepoProvider" yaml:"codeRepoProvider"`
	// +optional
	Product string `json:"product" yaml:"product"`
	// +optional
	Project string `json:"project" yaml:"project"`
	// +optional
	RepoName string `json:"repoName" yaml:"repoName"`
	// +optional
	URL string `json:"url,omitempty" yaml:"url"`
	// +optional
	DeploymentRuntime bool `json:"deploymentRuntime" yaml:"deploymentRuntime"`
	// +optional
	PipelineRuntime bool `json:"pipelineRuntime" yaml:"pipelineRuntime"`
	// +nullable
	// +optional
	Webhook *Webhook `json:"webhook" yaml:"webhook"`
}

// CodeRepoStatus defines the observed state of CodeRepo
type CodeRepoStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions" yaml:"conditions"`
	// +optional
	Sync2ArgoStatus *SyncCodeRepo2ArgoStatus `json:"sync2ArgoStatus,omitempty" yaml:"sync2ArgoStatus"`
}

type SyncCodeRepo2ArgoStatus struct {
	LastSuccessSpec string      `json:"lastSuccessSpec" yaml:"lastSuccessSpec"`
	LastSuccessTime metav1.Time `json:"lastSuccessTime" yaml:"lastSuccessTime"`
	Url             string      `json:"url" yaml:"url"`
	SecretID        string      `json:"secretID" yaml:"secretID"`
}

type CodeRepoConditionType string

func (status *CodeRepoStatus) GetConditions(conditionTypes map[string]bool) []metav1.Condition {
	result := make([]metav1.Condition, 0)
	for i := range status.Conditions {
		condition := status.Conditions[i]
		if ok := conditionTypes[condition.Type]; ok {
			result = append(result, condition)
		}

	}
	return result
}

func (status *CodeRepoStatus) SetConditions(conditions []metav1.Condition, evaluatedTypes map[string]bool) {
	appConditions := make([]metav1.Condition, 0)
	for i := 0; i < len(status.Conditions); i++ {
		condition := status.Conditions[i]
		if _, ok := evaluatedTypes[condition.Type]; !ok {
			appConditions = append(appConditions, condition)
		}
	}
	for i := range conditions {
		condition := conditions[i]
		eci := findConditionIndexByType(status.Conditions, condition.Type)
		if eci >= 0 &&
			status.Conditions[eci].Message == condition.Message &&
			status.Conditions[eci].Status == condition.Status &&
			status.Conditions[eci].Reason == condition.Reason {
			// If we already have a condition of this type, only update the timestamp if something
			// has changed.
			status.Conditions[eci].LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, status.Conditions[eci])
		} else {
			// Otherwise we use the new incoming condition with an updated timestamp:
			condition.LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, condition)
		}
	}
	sort.Slice(appConditions, func(i, j int) bool {
		left := appConditions[i]
		right := appConditions[j]
		return fmt.Sprintf("%s/%s/%v", left.Type, left.Message, left.LastTransitionTime) < fmt.Sprintf("%s/%s/%v", right.Type, right.Message, right.LastTransitionTime)
	})
	status.Conditions = appConditions
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="PROVIDER",type=string,JSONPath=".spec.codeRepoProvider"
//+kubebuilder:printcolumn:name="PRODUCTRESOURCENAME",type=string,JSONPath=".spec.product"
//+kubebuilder:printcolumn:name="PROJECT",type=string,JSONPath=".spec.project"
//+kubebuilder:printcolumn:name="REPONAME",type=string,JSONPath=".spec.repoName"
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=".spec.url"

// CodeRepo is the Schema for the coderepoes API
type CodeRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CodeRepoSpec   `json:"spec,omitempty"`
	Status CodeRepoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CodeRepoList contains a list of CodeRepo
type CodeRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CodeRepo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CodeRepo{}, &CodeRepoList{})
}

// Compare If true is returned, it means that the resource is duplicated
func (c *CodeRepo) Compare(obj client.Object) (bool, error) {
	return false, nil
}
