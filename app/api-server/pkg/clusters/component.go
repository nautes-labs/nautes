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

package cluster

import (
	"fmt"
	"html/template"
	"reflect"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type ComponentName string

type ComponentImplementation interface{}

type ComponentMap map[ComponentName]ComponentImplementation

type ComponentType string

type ComponentDefinition map[ComponentType]ComponentMap

type ComponentsList struct {
	Pipeline         map[ComponentName]Pipeline         `json:"pipeline"`
	Deployemnt       map[ComponentName]Deployment       `json:"deployment"`
	Gateway          map[ComponentName]Gateway          `json:"gateway"`
	OAuthProxy       map[ComponentName]OAuthProxy       `json:"oauthProxy"`
	SecretManagement map[ComponentName]SecretManagement `json:"secretManagement"`
}

type ComponentsListFunc func(interface{}) interface{}

func NewComponentsList() *ComponentsList {
	return &ComponentsList{
		Pipeline: map[ComponentName]Pipeline{
			"tekton": NewTekton(),
		},
		Deployemnt: map[ComponentName]Deployment{
			"argocd": NewArgocd(),
		},
		Gateway: map[ComponentName]Gateway{
			"traefik": NewTraefik(),
		},
		OAuthProxy: map[ComponentName]OAuthProxy{
			"oauth2-proxy": NewOAuth2Proxy(),
		},
		SecretManagement: map[ComponentName]SecretManagement{
			"vault": NewVault(),
		},
	}
}

// GetComponent retrieves a specific component by its type and componentName from the ComponentsList.
// The function uses reflection to iterate over the struct fields, searching for the field tagged with the componentType.
// Once found, it checks if the field is a map and then searches for the given componentName within that map.
// If successful, it returns the component's instance; otherwise, an error is thrown.
func (c *ComponentsList) GetComponent(componentType, componentName string) (interface{}, error) {
	componentsType := reflect.TypeOf(c).Elem()
	componentsValue := reflect.ValueOf(c).Elem()

	for i := 0; i < componentsType.NumField(); i++ {
		field := componentsType.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != componentType {
			continue
		}

		fieldValue := componentsValue.Field(i)
		if fieldValue.Kind() == reflect.Map {
			mapKeys := fieldValue.MapKeys()
			for _, key := range mapKeys {
				if key.String() == componentName {
					ins := fieldValue.MapIndex(key).Interface()
					return ins, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("'%s' component %s is not found", componentType, componentName)
}

// GenerateTemplateFuncs Find components in the component list.
// Find the implementation of specific components and call the method of generating template maps.
// Return maps of all components.
func (c *ComponentsList) GenerateTemplateFuncs(cluster *resourcev1alpha1.Cluster) ([]template.FuncMap, error) {
	var funcMaps []template.FuncMap

	componentTypes := reflect.TypeOf(cluster.Spec.ComponentsList)
	componentValues := reflect.ValueOf(cluster.Spec.ComponentsList)

	for i := 0; i < componentValues.NumField(); i++ {
		componentValue := componentValues.Field(i)
		if componentValue.IsNil() || !componentValue.IsValid() {
			continue
		}

		componentType := componentTypes.Field(i).Tag.Get("json")
		componentName := componentValue.Elem().FieldByName("Name").String()
		if componentName == "" {
			return nil, fmt.Errorf("failed to get %s component name", componentType)
		}
		component, err := c.GetComponent(componentType, componentName)
		if err != nil {
			return nil, err
		}

		registerTemplateFuncs := reflect.ValueOf(component).MethodByName("RegisterTemplateFuncs")
		if !registerTemplateFuncs.IsValid() {
			return nil, fmt.Errorf("failed to call 'RegisterTemplateFuncs' method")
		}
		results := registerTemplateFuncs.Call(nil)
		if len(results) == 0 {
			return nil, fmt.Errorf("failed to get result by 'RegisterTemplateFuncs' method")
		}

		funcMap, ok := results[0].Interface().(template.FuncMap)
		if !ok {
			return nil, fmt.Errorf("failed to get template func map")
		}

		funcMaps = append(funcMaps, funcMap)
	}

	return funcMaps, nil
}
