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

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	EnvNameRuntimeOperatorConfigPath = "RUNTIME_OPERATOR_CONFIG_PATH"
)

type RuntimeOperatorConfig struct {
	ProviderCallerMapping map[string]string `yaml:"providerCallerMapping,omitempty"`
}

func NewConfig() (*RuntimeOperatorConfig, error) {
	configPath := os.Getenv(EnvNameRuntimeOperatorConfigPath)
	var configByte []byte
	if configPath != "" {
		var err error
		configByte, err = os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	config := &RuntimeOperatorConfig{
		ProviderCallerMapping: map[string]string{
			"operatorhub": "http",
		},
	}

	if configByte != nil {
		if err := yaml.Unmarshal(configByte, config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return config, nil
}
