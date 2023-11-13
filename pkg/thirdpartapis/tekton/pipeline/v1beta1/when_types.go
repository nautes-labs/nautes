/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/selection"
)

// WhenExpression allows a PipelineTask to declare expressions to be evaluated before the Task is run
// to determine whether the Task should be executed or skipped
type WhenExpression struct {
	// Input is the string for guard checking which can be a static input or an output from a parent Task
	Input string `json:"input"`

	// Operator that represents an Input's relationship to the values
	Operator selection.Operator `json:"operator"`

	// Values is an array of strings, which is compared against the input, for guard checking
	// It must be non-empty

	Values []string `json:"values"`
}

// WhenExpressions are used to specify whether a Task should be executed or skipped
// All of them need to evaluate to True for a guarded Task to be executed.
type WhenExpressions []WhenExpression
