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

	"gopkg.in/yaml.v2"
)

func GetComponentDefinition(path string) (*UsageComponentDefinition, error) {
	content, err := loadComponentCategoryDefinition(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load component category definition, err: %w", err)
	}

	usageComponentDefinition := &UsageComponentDefinition{}
	err = yaml.Unmarshal([]byte(content), &usageComponentDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %w", err)
	}

	return usageComponentDefinition, nil
}

func GetClusterCommonConfig(path string) (*ClusterCommonConfig, error) {
	content, err := loadClusterCommonConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster common config, err: %w", err)
	}

	config := &ClusterCommonConfig{}
	err = yaml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal component type configuration, err: %w", err)
	}

	return config, nil
}

func loadComponentCategoryDefinition(path string) (string, error) {
	if val := os.Getenv(EnvComponentCategoryDefinition); val != "" {
		path = val
	}

	return loadConfigFile(path)
}

func loadClusterCommonConfig(path string) (string, error) {
	if val := os.Getenv(EnvClusterCommonConfig); val != "" {
		path = val
	}

	return loadConfigFile(path)
}

func loadConfigFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read configuration file: %w", err)
	}

	return string(bytes), nil
}
