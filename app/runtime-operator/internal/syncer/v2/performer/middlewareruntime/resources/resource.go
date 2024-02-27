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

type MetaData interface {
	// GetType returns the type of the resource, such as Deployment, Service, and so on.
	GetType() string
	// GetName returns the name of the resource.
	GetName() string
	// GetUniqueID returns the unique identifier of the resource.
	GetUniqueID() string
	// IsSensitiveResource indicates whether the resource is a sensitive resource.
	IsSensitiveResource() bool
}

type ResourceMetadata struct {
	Type   string            `json:"type" yaml:"type"`
	Name   string            `json:"name" yaml:"name"`
	Space  string            `json:"space,omitempty" yaml:"space,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	// SensitiveResource indicates whether the resource is a secret resource.
	// If the resource is a secret resource, it will remove sensitive information in the resource status.
	SensitiveResource bool `json:"isSensitiveResource" yaml:"isSensitiveResource"`
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

func (r ResourceMetadata) IsSensitiveResource() bool {
	return r.SensitiveResource
}

type Dependencies interface {
	// GetDependencies returns the dependencies of the resource.
	GetDependencies() []ResourceMetadata
}

type ResourceDependencies []ResourceMetadata

func (d ResourceDependencies) GetDependencies() []ResourceMetadata {
	return d
}

type Attributes interface {
	// GetAttributes returns the collection of attributes of the resource.
	GetAttributes() interface{}
	// ClearSensitiveAttributes clears the sensitive information in the resource.
	ClearSensitiveAttributes()
}

type Status interface {
	GetStatus() ResourceStatus
	SetStatus(status ResourceStatus)
	ClearSensitiveStatus()
}

type ResourceStatus struct {
	// Properties is a collection of properties of the resource.
	// It used to validate the resource is changed or not.
	Properties map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`
	// Raw is the status of the resource in raw format.
	Raw []byte `json:"raw,omitempty" yaml:"raw,omitempty"`
	// Peer is the status of the resource in the peer environment.
	// Current use cases:
	//   When updating a resource, additional parameters may be required, such as the resource's version number.
	//   Set this value using the Get method, and then retrieve it during the update.
	Peer map[string]string `json:"peer,omitempty" yaml:"peer,omitempty"`
}

func (rs *ResourceStatus) GetStatus() ResourceStatus {
	return *rs
}

func (rs *ResourceStatus) SetStatus(status ResourceStatus) {
	rs.Properties = status.Properties
	rs.Raw = status.Raw
	rs.Peer = status.Peer
}

func (rs *ResourceStatus) ClearSensitiveStatus() {
	rs.Raw = nil
}

// Resource is an abstraction of a resource, such as a Deployment, a Service, and so on.
// Resource represents a generic resource in the system.
type Resource interface {
	MetaData
	Dependencies
	Attributes
	Status
}

// CommonResource is a common resource that can be used to represent any resource type. It is mainly used for user-defined resource types.
type CommonResource struct {
	ResourceMetadata     `json:"metadata" yaml:"metadata"`
	ResourceDependencies `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Spec                 map[string]string `json:"spec,omitempty" yaml:"spec,omitempty"`
	SensitiveSpec        map[string]string `json:"sensitiveSpec,omitempty" yaml:"sensitiveSpec,omitempty"`
	Status               ResourceStatus    `json:"status,omitempty" yaml:"status,omitempty"`
}

func (cr *CommonResource) GetAttributes() interface{} {
	return cr.Spec
}

func (cr *CommonResource) ClearSensitiveAttributes() {
	cr.SensitiveSpec = nil
}

func (cr *CommonResource) GetStatus() ResourceStatus {
	return cr.Status
}

func (cr *CommonResource) SetStatus(status ResourceStatus) {
	cr.Status = status
}

func (cr *CommonResource) ClearSensitiveStatus() {
	cr.Status.ClearSensitiveStatus()
}
