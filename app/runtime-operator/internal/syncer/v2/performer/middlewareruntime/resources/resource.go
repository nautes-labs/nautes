// Copyright 2024 Nautes Authors
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

package resources

import "fmt"

type ResourceMetadata struct {
	Type string `json:"type" yaml:"type"`
	Name string `json:"name" yaml:"name"`
}

func (r ResourceMetadata) GetType() string {
	return r.Type
}

func (r ResourceMetadata) GetName() string {
	return r.Name
}

func (r ResourceMetadata) GetUniqueID() string {
	return fmt.Sprintf("%s/%s", r.GetName(), r.GetType())
}

type Dependencies []ResourceMetadata

func (d Dependencies) GetDependencies() []ResourceMetadata {
	return d
}

// Resource is an abstraction of a resource, such as a Deployment, a Service, and so on.
// Resource represents a generic resource in the system.
type Resource interface {
	// GetType returns the type of the resource, such as Deployment, Service, and so on.
	GetType() string
	// GetName returns the name of the resource.
	GetName() string
	// GetUniqueID returns the unique identifier of the resource.
	GetUniqueID() string
	// GetDependencies returns the dependencies of the resource.
	GetDependencies() []ResourceMetadata
	// GetResourceAttributes returns the collection of attributes of the resource.
	GetResourceAttributes() interface{}
	GetStatus() interface{}
	SetStatus(status interface{}) error
}

// CommonResource is a common resource that can be used to represent any resource type. It is mainly used for user-defined resource types.
type CommonResource struct {
	ResourceMetadata `json:"metadata" yaml:"metadata"`
	Dependencies     `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Spec             map[string]string `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status           map[string]string `json:"status,omitempty" yaml:"status,omitempty"`
}

func (cr *CommonResource) GetResourceAttributes() interface{} {
	return cr.Spec
}

func (cr *CommonResource) GetStatus() interface{} {
	return cr.Status
}

func (cr *CommonResource) SetStatus(status interface{}) error {
	if status == nil {
		return nil
	}

	newStatus, ok := status.(map[string]string)
	if !ok {
		return fmt.Errorf("invalid status type: %T", status)
	}

	cr.Status = newStatus
	return nil
}
