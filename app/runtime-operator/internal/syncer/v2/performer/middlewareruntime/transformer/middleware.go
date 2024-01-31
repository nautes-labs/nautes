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

package transformer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"github.com/nautes-labs/nautes/pkg/nautesconst"
	"gopkg.in/yaml.v3"
)

const (
	defaultMiddlewareTransformRulesDir = "./middleware-transform-rules"
)

func init() {
	nautesHome = os.Getenv(nautesconst.EnvNautesHome)
}

var (
	// middlewareTransformRules is a map of MiddlewareTransformRule.
	// The key is "providerName/middlewareType/implementation".
	middlewareTransformRules = map[string]MiddlewareTransformRule{}
	nautesHome               string
)

type loadMiddlewareTransformRulesOptions struct {
	TransformRulesRootPath string
}

type LoadMiddlewareTransformRulesOption func(*loadMiddlewareTransformRulesOptions)

func WithPathForLoadMiddlewareTransformRules(rootPath string) LoadMiddlewareTransformRulesOption {
	return func(o *loadMiddlewareTransformRulesOptions) {
		o.TransformRulesRootPath = rootPath
	}
}

// LoadMiddlewareTransformRules loads the middleware transform rules from the specified root path.
// It accepts optional LoadMiddlewareTransformRulesOption to customize the loading behavior.
// If the root path does not exist, it returns nil without any error.
// The middleware transform rules are loaded from files under the root path in the following format:
// Each file's content is loaded into a MiddlewareTransformRule and added to the collection of middleware transform rules.
// If any error occurs during the loading process, it returns the corresponding error.
func LoadMiddlewareTransformRules(opts ...LoadMiddlewareTransformRulesOption) error {
	options := &loadMiddlewareTransformRulesOptions{
		TransformRulesRootPath: path.Join(nautesHome, defaultMiddlewareTransformRulesDir),
	}
	for _, opt := range opts {
		opt(options)
	}

	if _, err := os.Stat(options.TransformRulesRootPath); os.IsNotExist(err) {
		return nil
	}

	return utils.LoadFilesInPath(options.TransformRulesRootPath, func(data []byte) error {
		rule := &MiddlewareTransformRule{}
		if err := yaml.Unmarshal(data, rule); err != nil {
			return fmt.Errorf("failed to unmarshal middleware transform rule: %w", err)
		}

		if err := AddMiddlewareTransformRule(*rule); err != nil {
			return fmt.Errorf("failed to add middleware transform rule: %w", err)
		}

		return nil
	})
}

// ClearMiddlewareTransformRule clears the middleware transform rules.
func ClearMiddlewareTransformRule() {
	middlewareTransformRules = map[string]MiddlewareTransformRule{}
}

func getMiddlewareTransformRuleKey(providerName, middlewareType, implementation string) string {
	return fmt.Sprintf("%s/%s/%s", providerName, middlewareType, implementation)
}

func checkMiddlewareTransformRuleExisted(ruleKey string) bool {
	_, ok := middlewareTransformRules[ruleKey]
	return ok
}

// GetMiddlewareTransformRule retrieves the middleware transform rule based on the provider name, middleware type, and implementation.
// It returns the MiddlewareTransformRule and an error if the rule is not found.
func GetMiddlewareTransformRule(providerName, middlewareType, implementation string) (*MiddlewareTransformRule, error) {
	ruleKey := getMiddlewareTransformRuleKey(providerName, middlewareType, implementation)
	if !checkMiddlewareTransformRuleExisted(ruleKey) {
		return nil, runtimeerr.ErrorTransformRuleNotFound(errors.New(ruleKey))
	}

	rule := middlewareTransformRules[ruleKey]

	return &rule, nil
}

// AddMiddlewareTransformRule adds a middleware transform rule to the collection.
// It takes a MiddlewareTransformRule as input and returns an error if the rule already exists.
// The rule is identified by the combination of provider type, middleware type, and implementation.
// If the rule already exists, it returns an error indicating that the transform rule already exists.
// Otherwise, it adds the rule to the collection and returns nil.
func AddMiddlewareTransformRule(rule MiddlewareTransformRule) error {
	ruleKey := getMiddlewareTransformRuleKey(rule.ProviderType, rule.MiddlewareType, rule.Implementation)
	if checkMiddlewareTransformRuleExisted(ruleKey) {
		return runtimeerr.ErrorTransformRuleExists(errors.New(ruleKey))
	}

	middlewareTransformRules[ruleKey] = rule
	return nil
}

// UpdateResourceTransformerRule updates the MiddlewareTransformRule for a given rule.
// It takes a MiddlewareTransformRule as input and returns an error if the rule is not found.
// The rule is identified by its ProviderType, MiddlewareType, and Implementation.
// If the rule is found, it updates the rule in the middlewareTransformRules map and returns nil.
// If the rule is not found, it returns an error indicating that the rule was not found.
func UpdateResourceTransformerRule(rule MiddlewareTransformRule) error {
	ruleKey := getMiddlewareTransformRuleKey(rule.ProviderType, rule.MiddlewareType, rule.Implementation)
	if !checkMiddlewareTransformRuleExisted(ruleKey) {
		return runtimeerr.ErrorTransformRuleNotFound(errors.New(ruleKey))
	}

	middlewareTransformRules[ruleKey] = rule
	return nil
}

// RemoveMiddlewareTransformRule removes a middleware transform rule based on the provided parameters.
// It takes the provider name, middleware type, and implementation as input.
// If the rule does not exist, it returns an error indicating that the transform rule was not found.
// Otherwise, it deletes the rule from the middlewareTransformRules map and returns nil.
func RemoveMiddlewareTransformRule(providerName, middlewareType, implementation string) error {
	ruleKey := getMiddlewareTransformRuleKey(providerName, middlewareType, implementation)
	if !checkMiddlewareTransformRuleExisted(ruleKey) {
		return runtimeerr.ErrorTransformRuleNotFound(errors.New(ruleKey))
	}

	delete(middlewareTransformRules, ruleKey)
	return nil
}

// MiddlewareTransformRule is a set of transformation rules for converting middleware to custom resources.
type MiddlewareTransformRule struct {
	// ProviderType is the provider type associated with the rule.
	ProviderType string `yaml:"providerType"`
	// MiddlewareType is the middleware type associated with the rule.
	MiddlewareType string `yaml:"middlewareType"`
	// Implementation is the implementation method of the middleware.
	Implementation string `yaml:"implementation"`
	// Resources is the custom resource templates corresponding to the middleware, using the format of the text/template package in golang.
	Resources []string `yaml:"resources"`
}

func (mtr *MiddlewareTransformRule) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 2 {
		switch value.Content[i].Value {
		case "providerType":
			mtr.ProviderType = value.Content[i+1].Value
		case "middlewareType":
			mtr.MiddlewareType = value.Content[i+1].Value
		case "implementation":
			mtr.Implementation = value.Content[i+1].Value
		case "resources":
			if len(value.Content[i+1].Value) == 0 {
				return fmt.Errorf("resources is empty")
			}

			mtr.Resources = []string{}
			resTemplates := strings.Split(value.Content[i+1].Value, "\n---\n")
			for _, res := range resTemplates {
				tmpl := strings.TrimSpace(res)
				if len(tmpl) == 0 {
					continue
				}
				mtr.Resources = append(mtr.Resources, tmpl)
			}
		}
	}

	if mtr.Implementation == "" {
		return fmt.Errorf("implementation is empty")
	}

	if mtr.MiddlewareType == "" {
		return fmt.Errorf("middleware type is empty")
	}

	if mtr.ProviderType == "" {
		return fmt.Errorf("provider type is empty")
	}

	if len(mtr.Resources) == 0 {
		return fmt.Errorf("resources is empty")
	}

	for i, resource := range mtr.Resources {
		if resource == "" {
			return fmt.Errorf("resource %d is empty", i)
		}
	}

	return nil
}

// ConvertMiddlewareToResources converts middleware to custom resource declarations.
func ConvertMiddlewareToResources(providerName string, middleware v1alpha1.Middleware) (res []resources.Resource, err error) {
	// Get the transform rule from middlewareTransformRules.
	rule, err := GetMiddlewareTransformRule(providerName, middleware.Type, middleware.Implementation)
	if err != nil {
		return nil, fmt.Errorf("failed to get middleware transform rule: %w", err)
	}

	var resArray []resources.Resource

	// Render the resource content based on the resource template, the content is in yaml format.
	for _, resource := range rule.Resources {
		// Use middleware to render the resource content.
		rendered, err := renderResource(resource, middleware)
		if err != nil {
			return nil, err
		}

		// Skip the empty content.
		if strings.TrimSpace(string(rendered)) == "" {
			continue
		}

		// Translate the rendered content into CommonResource objects.
		var commonResource resources.CommonResource
		err = yaml.Unmarshal(rendered, &commonResource)
		if err != nil {
			return nil, err
		}

		commonResource.Space = middleware.Space
		commonResource.Labels = middleware.Labels

		// Append the CommonResource to the resources slice.
		resArray = append(resArray, &commonResource)
	}

	// Return the resources.
	return resArray, nil
}

// renderResource renders a resource template with the provided middleware data.
// It parses the resource template, executes it with the middleware data, and returns the rendered resource as a byte slice.
// If there is an error during parsing or execution, it returns an error.
func renderResource(resource string, middleware v1alpha1.Middleware) ([]byte, error) {
	// Parse the resource template.
	tmpl, err := template.New("resource").Parse(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource template: %w", err)
	}

	// Execute the template with the middleware data.
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, middleware)
	if err != nil {
		return nil, fmt.Errorf("failed to execute resource template: %w", err)
	}

	// Return the rendered resource.
	return buf.Bytes(), nil
}
