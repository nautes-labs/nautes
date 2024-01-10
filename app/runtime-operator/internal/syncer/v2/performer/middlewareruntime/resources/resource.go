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

type ResourceMetadata struct {
	Type string
	Name string
}

func (r ResourceMetadata) GetType() string {
	return r.Type
}

func (r ResourceMetadata) GetName() string {
	return r.Name
}

// ResourceDependency represents a resource's dependency on other resources.
type ResourceDependency struct {
	// Type is the type of the dependency resource.
	Type string
	// Name is the name of the dependency resource.
	Name string
}

type Dependencies []ResourceDependency

func (d Dependencies) GetDependencies() []ResourceDependency {
	return d
}

// Resource is an abstraction of a resource, such as a Deployment, a Service, and so on.
type Resource interface {
	// GetType returns the type of the resource, such as Deployment, Service, and so on.
	GetType() string
	// GetName returns the name of the resource.
	GetName() string
	// GetDependencies returns the dependencies of the resource.
	GetDependencies() []ResourceDependency
	// GetResourceAttributes returns the collection of attributes of the resource.
	GetResourceAttributes() interface{}
}

// CommonResource is a common resource that can be used to represent any resource type. It is mainly used for user-defined resource types.
type CommonResource struct {
	ResourceMetadata
	Dependencies
	Spec map[string]string
}

func (cr *CommonResource) GetResourceAttributes() interface{} {
	return cr.Spec
}
