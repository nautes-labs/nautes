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
	LABEL_FROM_PRODUCT          = "resource.nautes.io/reference"
	LABEL_BELONG_TO_PRODUCT     = "resource.nautes.io/belongsto"
	LABEL_FROM_PRODUCT_PROVIDER = "resource.nautes.io/from"
)

const (
	EnvUseRealName = "USEREALNAME"
)

func GetConditions(conditions []metav1.Condition, conditionTypes map[string]bool) []metav1.Condition {
	result := make([]metav1.Condition, 0)
	for i := range conditions {
		condition := conditions[i]
		if ok := conditionTypes[condition.Type]; ok {
			result = append(result, condition)
		}

	}
	return result
}

func GetNewConditions(src, conditions []metav1.Condition, evaluatedTypes map[string]bool) []metav1.Condition {
	appConditions := make([]metav1.Condition, 0)
	for i := 0; i < len(src); i++ {
		condition := src[i]
		if _, ok := evaluatedTypes[condition.Type]; !ok {
			appConditions = append(appConditions, condition)
		}
	}
	for i := range conditions {
		condition := conditions[i]
		eci := findConditionIndexByType(src, condition.Type)
		if eci >= 0 &&
			src[eci].Message == condition.Message &&
			src[eci].Status == condition.Status &&
			src[eci].Reason == condition.Reason {
			// If we already have a condition of this type, only update the timestamp if something
			// has changed.
			src[eci].LastTransitionTime = metav1.Now()
			appConditions = append(appConditions, src[eci])
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
	return appConditions
}

func findConditionIndexByType(conditions []metav1.Condition, t string) int {
	for i := range conditions {
		if conditions[i].Type == t {
			return i
		}
	}
	return -1
}

// +kubebuilder:object:generate=false
type Runtime interface {
	client.Object
	GetProduct() string
	GetName() string
	GetDestination() string
	GetNamespaces() []string
}

func convertArrayToBoolMap(in []string) map[string]bool {
	out := map[string]bool{}
	for _, str := range in {
		out[str] = true
	}
	return out
}
