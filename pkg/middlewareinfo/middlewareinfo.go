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

package middlewareinfo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nautes-labs/nautes/pkg/nautesenv"
	"gopkg.in/yaml.v2"
)

func init() {
	metadatas, err := NewMiddlewares()
	if err != nil {
		panic(err)
	}
	MiddlewareMetadata = *metadatas
}

var MiddlewareMetadata Middlewares

type Middleware struct {
	Name                    string              `json:"name" yaml:"name"`
	HasAuthenticationSystem bool                `json:"hasAuthenticationSystem" yaml:"hasAuthenticationSystem"`
	DefaultAuthType         string              `json:"defaultAuthType" yaml:"defaultAuthType"`
	AvailableAuthTypes      map[string]AuthType `json:"availableAuthTypes" yaml:"availableAuthTypes"`
	Providers               map[string]Provider `json:"providers" yaml:"providers"`
}

type AuthType struct {
	Type string `json:"type" yaml:"type"`
	// TODO: Provide password generation requirements
}

type Provider struct {
	Name string `json:"name" yaml:"name"`
	// DefaultImplementation is implementation should be used by default.
	DefaultImplementation string                              `json:"defaultImplementation" yaml:"defaultImplementation"`
	Implementations       map[string]MiddlewareImplementation `json:"implementations" yaml:"implementations"`
}

type MiddlewareImplementation struct {
	Name string `json:"name" yaml:"name"`
	// IntroductionPages is a map of URLs about the middleware.
	// The key is the type of the page, such as "homepage", "documentation", "github".
	// The value is the URL.
	IntroductionPages map[string]string `json:"introductionPages,omitempty" yaml:"introductionPages,omitempty"`
	// AvailableVars is the variables that the implementation can use.
	AvailableVars []string `json:"availableVars,omitempty" yaml:"availableVars,omitempty"`
}

func WithLoadPath(loadPath string) NewMiddlewaresOption {
	return func(o *newMiddlewaresOptions) {
		o.LoadPath = loadPath
	}
}

const defaultLoadPath = "./config/middlewares.yaml"

type newMiddlewaresOptions struct{ LoadPath string }
type NewMiddlewaresOption func(*newMiddlewaresOptions)
type Middlewares map[string]Middleware

func NewMiddlewares(opts ...NewMiddlewaresOption) (*Middlewares, error) {
	options := &newMiddlewaresOptions{
		LoadPath: filepath.Join(nautesenv.GetNautesHome(), defaultLoadPath),
	}
	for _, o := range opts {
		o(options)
	}

	middlewares := make(Middlewares)
	middlewaresByte, err := os.ReadFile(options.LoadPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &middlewares, nil
		}
		return nil, fmt.Errorf("failed to read middlewares file: %w", err)
	}

	if err := yaml.Unmarshal(middlewaresByte, &middlewares); err != nil {
		return nil, fmt.Errorf("failed to unmarshal middlewares: %w", err)
	}

	// Check required fields and fill in missing fields
	for middlewareName, middleware := range middlewares {
		for authTypeName, authType := range middleware.AvailableAuthTypes {
			authType.Type = authTypeName
			middleware.AvailableAuthTypes[authTypeName] = authType
		}

		for providerName, provider := range middleware.Providers {
			for implementationName, implementation := range provider.Implementations {
				implementation.Name = implementationName
				provider.Implementations[implementationName] = implementation
			}

			provider.Name = providerName
			middleware.Providers[providerName] = provider
		}

		middleware.Name = middlewareName
		middlewares[middlewareName] = middleware
	}

	if err := middlewares.Verify(); err != nil {
		return nil, fmt.Errorf("failed to verify middlewares: %w", err)
	}

	return &middlewares, nil
}

func (m *Middlewares) Verify() error {
	for middlewareName, middleware := range *m {
		if middleware.HasAuthenticationSystem {
			if len(middleware.AvailableAuthTypes) == 0 {
				return fmt.Errorf("available auth types in middleware %s is empty", middlewareName)
			}

			_, ok := middleware.AvailableAuthTypes[middleware.DefaultAuthType]
			if !ok {
				return fmt.Errorf("default auth type %s in middleware %s does not exist", middleware.DefaultAuthType, middlewareName)
			}
		}

		for providerName, provider := range middleware.Providers {
			if provider.DefaultImplementation == "" {
				return fmt.Errorf("default implementation in provider %s of middleware %s is empty", providerName, middlewareName)
			}

			if _, ok := provider.Implementations[provider.DefaultImplementation]; !ok {
				return fmt.Errorf("default implementation %s in provider %s of middleware %s does not exist", provider.DefaultImplementation, providerName, middlewareName)
			}
		}
	}
	return nil
}

// IsAuthTypeSupported checks if the access info type is supported by the middleware.
func (m *Middleware) IsAuthTypeSupported(accessInfoType string) bool {
	_, ok := m.AvailableAuthTypes[accessInfoType]
	return ok
}
