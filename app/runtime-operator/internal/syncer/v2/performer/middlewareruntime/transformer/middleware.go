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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
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

// parseFilePathMiddlewareTransformRule parses the given rule file path and extracts the provider type, middleware type, and implementation.
// It expects the rule file path to be in the format "/providerType/middlewareType/implementation.ext".
// The rootPathSegmentNum parameter specifies the number of root path segments before the provider type.
// It returns the provider type, middleware type, implementation, and an error if the path format is invalid.
func parseFilePathMiddlewareTransformRule(ruleFilePath string, rootPathSegmentNum int) (
	providerType string,
	middlewareType string,
	implementation string,
	err error) {
	segments := strings.Split(ruleFilePath, "/")
	if len(segments)-rootPathSegmentNum != 3 {
		return "", "", "", fmt.Errorf("invalid path format: %s", ruleFilePath)
	}

	providerType = segments[len(segments)-3]
	middlewareType = segments[len(segments)-2]
	implementation = strings.TrimSuffix(segments[len(segments)-1], filepath.Ext(segments[len(segments)-1]))

	return providerType, middlewareType, implementation, nil
}

// LoadMiddlewareTransformRules loads the middleware transform rules from the specified root path.
// It accepts optional LoadMiddlewareTransformRulesOption to customize the loading behavior.
// If the root path does not exist, it returns nil without any error.
// The middleware transform rules are loaded from files under the root path in the following format:
// ".path -> providerName -> middlewareType -> implementation".
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

	rootPathSegmentNum := len(strings.Split(options.TransformRulesRootPath, "/"))

	// Load all the files under middlewareTransformRulesDir, path style ".path -> providerName -> middlewareType -> implementation".
	err := filepath.Walk(options.TransformRulesRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		providerName, middlewareType, implementation, err := parseFilePathMiddlewareTransformRule(path, rootPathSegmentNum)
		if err != nil {
			return fmt.Errorf("failed to parse file path: %w", err)
		}

		// For each file, load the content into middlewareTransformRules.
		ruleByte, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		rule, err := NewMiddlewareTransformRule(providerName, middlewareType, implementation, ruleByte)
		if err != nil {
			return fmt.Errorf("failed to create middleware transform rule: %w", err)
		}

		if err = AddMiddlewareTransformRule(*rule); err != nil {
			return fmt.Errorf("failed to add middleware transform rule: %w", err)
		}

		return nil
	})

	return err
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
	ProviderType string
	// MiddlewareType is the middleware type associated with the rule.
	MiddlewareType string
	// Implementation is the implementation method of the middleware.
	Implementation string
	// Resources is the custom resource templates corresponding to the middleware, using the format of the text/template package in golang.
	Resources []string
}

// NewMiddlewareTransformRule creates a new instance of MiddlewareTransformRule.
// It takes the providerType, middlewareType, implementation, and ruleByte as input parameters.
// The ruleByte is split into multiple resource byte arrays using the delimiter "\n---\n".
// Each resource string is then converted into a byte array.
// The function returns a pointer to the newly created MiddlewareTransformRule and an error, if any.
func NewMiddlewareTransformRule(providerType, middlewareType, implementation string, ruleByte []byte) (*MiddlewareTransformRule, error) {
	// Split ruleByte into many resource byte arrays using a delimiter.
	resourceBytes := bytes.Split(ruleByte, []byte("\n---\n"))

	// Convert each resource string into a byte array.
	var resArray []string
	for _, resourceByte := range resourceBytes {
		resArray = append(resArray, strings.TrimSpace(string(resourceByte)))
	}

	// Create MiddlewareTransformRule and return.
	return &MiddlewareTransformRule{
		ProviderType:   providerType,
		MiddlewareType: middlewareType,
		Implementation: implementation,
		Resources:      resArray,
	}, nil
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

		// Translate the rendered content into CommonResource objects.
		var commonResource resources.CommonResource
		err = yaml.Unmarshal(rendered, &commonResource)
		if err != nil {
			return nil, err
		}

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
