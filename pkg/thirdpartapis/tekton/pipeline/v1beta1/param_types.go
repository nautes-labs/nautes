/*
Copyright 2019 The Tekton Authors

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
	"encoding/json"
	"fmt"

	resource "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

// ParamsPrefix is the prefix used in $(...) expressions referring to parameters
const ParamsPrefix = "params"

// ParamSpec defines arbitrary parameters needed beyond typed inputs (such as
// resources). Parameter values are provided by users as inputs on a TaskRun
// or PipelineRun.
type ParamSpec struct {
	// Name declares the name by which a parameter is referenced.
	Name string `json:"name"`
	// Type is the user-specified type of the parameter. The possible types
	// are currently "string", "array" and "object", and "string" is the default.

	Type ParamType `json:"type,omitempty"`
	// Description is a user-facing description of the parameter that may be
	// used to populate a UI.

	Description string `json:"description,omitempty"`
	// Properties is the JSON Schema properties to support key-value pairs parameter.

	Properties map[string]PropertySpec `json:"properties,omitempty"`
	// Default is the value a parameter takes if no input value is supplied. If
	// default is set, a Task may be executed without a supplied value for the
	// parameter.

	Default *ArrayOrString `json:"default,omitempty"`
}

// PropertySpec defines the struct for object keys
type PropertySpec struct {
	Type ParamType `json:"type,omitempty"`
}

// ResourceParam declares a string value to use for the parameter called Name, and is used in
// the specific context of PipelineResources.
type ResourceParam = resource.ResourceParam

// Param declares an ArrayOrString to use for the parameter called name.
type Param struct {
	Name  string        `json:"name"`
	Value ArrayOrString `json:"value"`
}

// ParamType indicates the type of an input parameter;
// Used to distinguish between a single string and an array of strings.
type ParamType string

// Valid ParamTypes:
const (
	ParamTypeString ParamType = "string"
	ParamTypeArray  ParamType = "array"
	ParamTypeObject ParamType = "object"
)

// AllParamTypes can be used for ParamType validation.
var AllParamTypes = []ParamType{ParamTypeString, ParamTypeArray, ParamTypeObject}

// ArrayOrString is modeled after IntOrString in kubernetes/apimachinery:

// ArrayOrString is a type that can hold a single string or string array.
// Used in JSON unmarshalling so that a single JSON field can accept
// either an individual string or an array of strings.
// TODO (@chuangw6): This struct will be renamed or be embedded in a new struct to take into
// consideration the object case after the community reaches an agreement on it.
type ArrayOrString struct {
	Type      ParamType `json:"type"` // Represents the stored type of ArrayOrString.
	StringVal string    `json:"stringVal"`

	ArrayVal  []string          `json:"arrayVal"`
	ObjectVal map[string]string `json:"objectVal"`
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (arrayOrString *ArrayOrString) UnmarshalJSON(value []byte) error {
	// ArrayOrString is used for Results Value as well, the results can be any kind of
	// data so we need to check if it is empty.
	if len(value) == 0 {
		arrayOrString.Type = ParamTypeString
		return nil
	}
	if value[0] == '[' {
		// We're trying to Unmarshal to []string, but for cases like []int or other types
		// of nested array which we don't support yet, we should continue and Unmarshal
		// it to String. If the Type being set doesn't match what it actually should be,
		// it will be captured by validation in reconciler.
		// if failed to unmarshal to array, we will convert the value to string and marshal it to string
		var a []string
		if err := json.Unmarshal(value, &a); err == nil {
			arrayOrString.Type = ParamTypeArray
			arrayOrString.ArrayVal = a
			return nil
		}
	}
	if value[0] == '{' {
		// if failed to unmarshal to map, we will convert the value to string and marshal it to string
		var m map[string]string
		if err := json.Unmarshal(value, &m); err == nil {
			arrayOrString.Type = ParamTypeObject
			arrayOrString.ObjectVal = m
			return nil
		}
	}

	// By default we unmarshal to string
	arrayOrString.Type = ParamTypeString
	if err := json.Unmarshal(value, &arrayOrString.StringVal); err == nil {
		return nil
	}
	arrayOrString.StringVal = string(value)

	return nil
}

// MarshalJSON implements the json.Marshaller interface.
func (arrayOrString ArrayOrString) MarshalJSON() ([]byte, error) {
	switch arrayOrString.Type {
	case ParamTypeString:
		return json.Marshal(arrayOrString.StringVal)
	case ParamTypeArray:
		return json.Marshal(arrayOrString.ArrayVal)
	case ParamTypeObject:
		return json.Marshal(arrayOrString.ObjectVal)
	default:
		return []byte{}, fmt.Errorf("impossible ArrayOrString.Type: %q", arrayOrString.Type)
	}
}

// NewArrayOrString creates an ArrayOrString of type ParamTypeString or ParamTypeArray, based on
// how many inputs are given (>1 input will create an array, not string).
func NewArrayOrString(value string, values ...string) *ArrayOrString {
	if len(values) > 0 {
		return &ArrayOrString{
			Type:     ParamTypeArray,
			ArrayVal: append([]string{value}, values...),
		}
	}
	return &ArrayOrString{
		Type:      ParamTypeString,
		StringVal: value,
	}
}
