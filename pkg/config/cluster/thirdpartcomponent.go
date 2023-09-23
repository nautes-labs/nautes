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
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
)

// Configuration information of third-party components required for cluster registration.
type ThridPartComponent struct {
	Name        string      `yaml:"name"`
	Namespace   string      `yaml:"namespace"`
	Type        string      `yaml:"type"`
	Default     bool        `yaml:"default"`
	General     bool        `yaml:"general"`
	Properties  []Propertie `yaml:"properties"`
	InstallPath InstallPath `yaml:"installPath"`
}

type InstallPath struct {
	HostPhysical   []string `yaml:"host&physical"`
	WorkerPhysical []string `yaml:"worker&physical"`
	WorkerVirtual  []string `yaml:"worker&virtual"`
	Generic        []string `yaml:"generic"`
}

func (t *ThridPartComponent) GetInstallPath(key string) []string {
	mas := make(map[string][]string)
	val := reflect.ValueOf(t.InstallPath)

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := val.Type().Field(i).Tag.Get("yaml")

		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String {
			mas[tag] = field.Interface().([]string)
		}
	}

	return mas[key]
}

func GetThirdPartComponentsList(path string) ([]*ThridPartComponent, error) {
	content, err := loadThirdPartComponentsList(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load third-part components list, err: %s", err)
	}

	components := []*ThridPartComponent{}
	err = yaml.Unmarshal([]byte(content), &components)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal third part component configuration, err: %s", err)
	}

	for _, component := range components {
		if !componentTypeMap[component.Type] {
			return nil, fmt.Errorf("there is no component type for %s", component.Type)
		}
	}

	return components, nil
}

func loadThirdPartComponentsList(path string) (string, error) {
	if val := os.Getenv(EnvthirdPartComponents); val != "" {
		path = val
	}

	return loadConfigFile(path)
}
