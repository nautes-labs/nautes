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

package nodestree

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name     string   `json:"name" yaml:"name"`
	Kind     string   `json:"kind" yaml:"kind"`
	Optional bool     `json:"optional" yaml:"optional"`
	Count    int      `json:"count" yaml:"count"`
	Level    int      `json:"level" yaml:"level"`
	Sub      []Config `json:"sub" yaml:"sub"`
}

// NewConfig generate resources layout config
func NewConfig() (*Config, error) {
	var config = &Config{}
	layoutFilePath := os.Getenv("RESOURCES_LAYOUT")
	if layoutFilePath == "" {
		return nil, fmt.Errorf("the resource layout file is not found")
	}

	bytes, err := os.ReadFile(layoutFilePath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
