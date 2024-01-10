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

package middlewareruntime

import (
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/caller/http"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
)

// ConvertMiddlewareToResources converts middleware to custom resource declarations.
func ConvertMiddlewareToResources(providerType, implementation string, middleware v1alpha1.Middleware) (res []resources.Resource, err error) {
	// 1. Get the template from the template directory based on providerType and middleware.Type.
	// 2. Render the template with the middleware.
	// 3. Translate the rendered file into resource declarations.
	// 4. Return the resource declarations.
	return
}

// ConvertResourcesToAdvanceCallerResources converts custom resource declarations to resources recognized by AdvancedCaller.
func ConvertResourcesToAdvanceCallerResources(providerType, callerType, implementation, middlewareType string, res []resources.Resource) (callerRes string, opts map[string]interface{}, err error) {
	// 1. Get the template from the template directory based on providerType, callerType, and middlewareType.
	// 2. Render the template with the resource declarations.
	// 3. Return the template.
	return
}

// ResourceTransformers stores the transformers that convert CRUD operations of custom resources
// into transformers recognized by basic callers. Array format map[ProviderType][CallerType][ResourceType]
var ResourceTransformers map[string]map[string]map[string]ResourceTransformer

func GetResourceTransformer(providerType, callerType, resourceType string) (*ResourceTransformer, error) {
	if _, ok := ResourceTransformers[providerType]; !ok {
		return nil, fmt.Errorf("provider type %s not supported", providerType)
	}

	if _, ok := ResourceTransformers[providerType][callerType]; !ok {
		return nil, fmt.Errorf("caller type %s not supported", callerType)
	}

	if _, ok := ResourceTransformers[providerType][callerType][resourceType]; !ok {
		return nil, fmt.Errorf("resource type %s not supported", resourceType)
	}

	transformer := ResourceTransformers[providerType][callerType][resourceType]
	return &transformer, nil
}

type ResourceTransformer struct {
	Create RequestTransformer
	Get    RequestTransformer
	Update RequestTransformer
	Delete RequestTransformer
}

type BasicCallerType string

const (
	BasicCallerTypeHTTP BasicCallerType = "http"
)

// RequestTransformer provides message body transformation for a single resource of a single type of request and response parsing.
type RequestTransformer struct {
	// Type is the caller type that implements RequestTransformer.
	Type BasicCallerType
	HTTP RequestTransformerHTTP
}

func NewRequestTransformer(transformer interface{}) (rt RequestTransformer, err error) {
	switch tf := transformer.(type) {
	case RequestTransformerHTTP:
		rt.Type = BasicCallerTypeHTTP
		rt.HTTP = tf
	default:
		err = fmt.Errorf("unknown transformer")
	}
	return
}

func (rt *RequestTransformer) GenerateRequest(resource resources.Resource) (req interface{}, err error) {
	switch rt.Type {
	case BasicCallerTypeHTTP:
		return rt.HTTP.GenerateRequest(resource)
	default:
		err = fmt.Errorf("caller type %s not supported", rt.Type)
	}
	return
}

func (rt *RequestTransformer) ParseResponse(response string) (state map[string]string, err error) {
	switch rt.Type {
	case BasicCallerTypeHTTP:
		return rt.HTTP.ParseResponse(response)
	default:
		err = fmt.Errorf("caller type %s not supported", rt.Type)
	}
	return
}

type RequestTransformerHTTP interface {
	GenerateRequest(resource resources.Resource) (req *http.RequestHTTP, err error)
	ParseResponse(response string) (state map[string]string, err error)
}
