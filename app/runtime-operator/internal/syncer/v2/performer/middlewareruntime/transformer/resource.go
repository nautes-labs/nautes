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
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/caller/http"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	runtimeerr "github.com/nautes-labs/nautes/app/runtime-operator/pkg/error"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"gopkg.in/yaml.v3"
)

func init() {
	resourceTransformers = make(map[string]ResourceTransformer)
}

// resourceTransformers stores the transformers that convert CRUD operations of custom resources
// into transformers recognized by basic callers.
// The key is "providerName/callerType/resourceType".
var resourceTransformers map[string]ResourceTransformer

type loadResourceTransformsOptions struct {
	TransformRulesRootPath string
}

type LoadResourceTransformOption func(*loadResourceTransformsOptions)

func WithTransformRulesFilePath(filePath string) LoadResourceTransformOption {
	return func(options *loadResourceTransformsOptions) {
		options.TransformRulesRootPath = filePath
	}
}

const (
	DefaultTransformRulesRootPath = "./resource-transform-rules"
)

// parseFilePathResourceTransformRule parses the given rule file path and extracts
// the provider type, caller type, and resource type. It also validates the path format.
// It returns the extracted values or an error if the path format is invalid.
func parseFilePathResourceTransformRule(ruleFilePath string, rootPathSegmentNum int) (
	providerType string,
	callerType string,
	resourceType string,
	err error) {
	segments := strings.Split(ruleFilePath, "/")
	if len(segments)-rootPathSegmentNum != 3 {
		return "", "", "", fmt.Errorf("invalid path format: %s", ruleFilePath)
	}

	providerType = segments[len(segments)-3]
	callerType = segments[len(segments)-2]
	resourceType = strings.TrimSuffix(segments[len(segments)-1], filepath.Ext(segments[len(segments)-1]))

	return providerType, callerType, resourceType, nil
}

// LoadResourceTransformers loads and applies resource transformers based on the provided options.
// It walks through the transform rules directory, reads each transform rule file, and adds the corresponding transformer.
// If the directory does not exist, the function returns nil without any error.
// Each transform rule file should follow the path style ".path -> providerName -> callerType -> resourceType".
// The function returns an error if there is any issue with parsing, reading, unmarshaling, or adding the transform rules.
func LoadResourceTransformers(opt ...LoadResourceTransformOption) error {
	options := &loadResourceTransformsOptions{
		TransformRulesRootPath: path.Join(nautesHome, DefaultTransformRulesRootPath),
	}
	for _, o := range opt {
		o(options)
	}

	if _, err := os.Stat(options.TransformRulesRootPath); os.IsNotExist(err) {
		return nil
	}

	rootPathSegmentNum := len(strings.Split(options.TransformRulesRootPath, "/"))

	// Look for transform rules in the transform rules directory.
	// path style ".path -> providerName -> callerType -> resourceType".
	err := filepath.Walk(options.TransformRulesRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		provider, caller, resource, err := parseFilePathResourceTransformRule(path, rootPathSegmentNum)
		if err != nil {
			return fmt.Errorf("failed to parse transform rule file path: %v", err)
		}

		// Read the file
		ruleByte, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read transform rule file: %v", err)
		}

		// Create a ResourceTransformer from the file
		transformer := ResourceTransformer{}
		if err = yaml.Unmarshal(ruleByte, &transformer); err != nil {
			return fmt.Errorf("failed to unmarshal transform rules: %v", err)
		}

		if caller != transformer.CallerType {
			return fmt.Errorf("caller type %s in transform rules does not match the caller type %s in the file path", transformer.CallerType, caller)
		}

		transformer.ProviderType = provider
		transformer.ResourceType = resource

		if err := AddResourceTransformer(transformer); err != nil {
			return fmt.Errorf("failed to add transform: %v", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// ClearResourceTransformer clears the resource transformers map.
func ClearResourceTransformer() {
	resourceTransformers = make(map[string]ResourceTransformer)
}

func getResourceTransformerKey(providerName, callerType, resourceType string) string {
	return fmt.Sprintf("%s/%s/%s", providerName, callerType, resourceType)
}

func checkResourceTransformerExisted(ruleKey string) bool {
	_, ok := resourceTransformers[ruleKey]
	return ok
}

// GetResourceTransformer retrieves the ResourceTransformer for the given provider name, caller type, and resource type.
// It returns the ResourceTransformer if it exists, otherwise it returns an error.
func GetResourceTransformer(providerName, callerType, resourceType string) (*ResourceTransformer, error) {
	ruleKey := getResourceTransformerKey(providerName, callerType, resourceType)
	if !checkResourceTransformerExisted(ruleKey) {
		return nil, runtimeerr.ErrorTransformRuleNotFound(errors.New(ruleKey))
	}

	rule := resourceTransformers[ruleKey]
	return &rule, nil
}

// AddResourceTransformer adds a resource transformer rule to the collection.
// It takes a rule of type ResourceTransformer as a parameter.
// The rule is identified by a unique key generated from the provider type, caller type, and resource type.
// If a rule with the same key already exists, it returns an error.
// Otherwise, it adds the rule to the collection and returns nil.
func AddResourceTransformer(rule ResourceTransformer) error {
	ruleKey := getResourceTransformerKey(rule.ProviderType, rule.CallerType, rule.ResourceType)
	if checkResourceTransformerExisted(ruleKey) {
		return runtimeerr.ErrorTransformRuleExists(errors.New(ruleKey))
	}

	resourceTransformers[ruleKey] = rule
	return nil
}

// UpdateResourceTransformer updates the resource transformer rule with the given parameters.
// It checks if the rule exists and then updates it in the resourceTransformers map.
// If the rule does not exist, it returns an error indicating that the rule was not found.
func UpdateResourceTransformer(rule ResourceTransformer) error {
	ruleKey := getResourceTransformerKey(rule.ProviderType, rule.CallerType, rule.ResourceType)
	if !checkResourceTransformerExisted(ruleKey) {
		return runtimeerr.ErrorTransformRuleNotFound(errors.New(ruleKey))
	}

	resourceTransformers[ruleKey] = rule
	return nil
}

// RemoveResourceTransformer removes a resource transformer based on the provided parameters.
// It takes the provider name, caller type, and resource type as input and returns an error if the resource transformer is not found.
// If the resource transformer is found, it is deleted from the resourceTransformers map.
func RemoveResourceTransformer(providerName, callerType, resourceType string) error {
	ruleKey := getResourceTransformerKey(providerName, callerType, resourceType)
	if !checkResourceTransformerExisted(ruleKey) {
		return runtimeerr.ErrorResourceNotFound(errors.New(ruleKey))
	}

	delete(resourceTransformers, ruleKey)
	return nil
}

// ResourceTransformer is a collection of RequestTransformer that converts CRUD operations of custom resources
type ResourceTransformer struct {
	// ProviderType is the provider type that implements RequestTransformer.
	ProviderType string
	// ResourceType is the resource type that implements RequestTransformer.
	ResourceType string
	// CallerType is the caller type that implements RequestTransformer.
	CallerType string `json:"callerType" yaml:"callerType"`
	// Create is the transformer for creating a resource.
	Create RequestTransformerInterface `json:"create" yaml:"create"`
	// Get is the transformer for getting a resource.
	Get RequestTransformerInterface `json:"get" yaml:"get"`
	// Update is the transformer for updating a resource.
	Update RequestTransformerInterface `json:"update" yaml:"update"`
	// Delete is the transformer for deleting a resource.
	Delete RequestTransformerInterface `json:"delete" yaml:"delete"`
}

func (r *ResourceTransformer) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content); i += 2 {
		switch value.Content[i].Value {
		case "callerType":
			r.CallerType = value.Content[i+1].Value
		case "create":
			transformRule, err := yaml.Marshal(value.Content[i+1])
			if err != nil {
				return err
			}
			rt, err := NewRequestTransformer(r.CallerType, transformRule)
			if err != nil {
				return err
			}
			r.Create = rt
		case "get":
			transformRule, err := yaml.Marshal(value.Content[i+1])
			if err != nil {
				return err
			}
			rt, err := NewRequestTransformer(r.CallerType, transformRule)
			if err != nil {
				return err
			}
			r.Get = rt
		case "update":
			transformRule, err := yaml.Marshal(value.Content[i+1])
			if err != nil {
				return err
			}
			rt, err := NewRequestTransformer(r.CallerType, transformRule)
			if err != nil {
				return err
			}
			r.Update = rt
		case "delete":
			transformRule, err := yaml.Marshal(value.Content[i+1])
			if err != nil {
				return err
			}
			rt, err := NewRequestTransformer(r.CallerType, transformRule)
			if err != nil {
				return err
			}
			r.Delete = rt
		}
	}

	if r.CallerType == "" {
		return fmt.Errorf("callerType is empty")
	}
	return nil
}

// RequestTransformerInterface provides message body transformation for a single resource of a single type of request and response parsing.
type RequestTransformerInterface interface {
	// GenerateRequest generates a request for the given resource.
	// Input:
	// - resource: the resource to generate the request for.
	// Output:
	// - req: the generated request, that caller can use to send to the provider.
	// - err: error if any
	GenerateRequest(resource resources.Resource) (req interface{}, err error)
	// ParseResponse parses the response from the provider and returns the state of the resource.
	// Input:
	// - response: the response from the provider.
	// Output:
	// - state: the state of the resource.
	// - err: error if any
	ParseResponse(response []byte) (state map[string]string, err error)
}

// RequestTransformer provides message body transformation for a single resource of a single type of request and response parsing.
type RequestTransformer struct {
	// CallerType is the caller type that implements RequestTransformer.
	CallerType string
	// HTTP is the transformer for HTTP caller.
	HTTP RequestTransformerHTTP
}

// NewRequestTransformer creates a RequestTransformer based on the caller type and the transform rule.
// It returns an error if the caller type is not supported.
func NewRequestTransformer(callerType string, transformRule []byte) (rt *RequestTransformer, err error) {
	rt = &RequestTransformer{
		CallerType: callerType,
	}
	switch callerType {
	case component.CallerTypeHTTP:
		transformer, err := NewRequestTransformerHTTP(transformRule)
		if err != nil {
			return nil, fmt.Errorf("failed to create http transformer: %v", err)
		}
		rt.HTTP = *transformer
	default:
		return nil, fmt.Errorf("caller %s not supported", callerType)
	}
	return rt, err
}

func (rt *RequestTransformer) GenerateRequest(resource resources.Resource) (req interface{}, err error) {
	switch rt.CallerType {
	case component.CallerTypeHTTP:
		return rt.HTTP.GenerateRequest(resource)
	default:
		err = fmt.Errorf("caller %s not supported", rt.CallerType)
	}
	return
}

func (rt *RequestTransformer) ParseResponse(response []byte) (state map[string]string, err error) {
	switch rt.CallerType {
	case component.CallerTypeHTTP:
		return rt.HTTP.ParseResponse(response)
	default:
		err = fmt.Errorf("caller %s not supported", rt.CallerType)
	}
	return
}

type RequestTransformerHTTP struct {
	// TransformRule is the transform rule for HTTP caller.
	TransformRule http.RequestTransformerRule
}

func NewRequestTransformerHTTP(rule []byte) (*RequestTransformerHTTP, error) {
	transformRule := &http.RequestTransformerRule{}
	err := yaml.Unmarshal(rule, transformRule)
	if err != nil {
		return nil, err
	}
	return &RequestTransformerHTTP{
		TransformRule: *transformRule,
	}, nil
}

// GenerateRequest generates a request based on the provided resource.
func (rt *RequestTransformerHTTP) GenerateRequest(res resources.Resource) (req *http.RequestHTTP, err error) {
	if res == nil {
		return nil, fmt.Errorf("invalid input: resource is nil")
	}

	// Render the request body using templates and resource definition
	var body *string
	if rt.TransformRule.RequestGenerationRule.Body != nil {
		bodyTemplate, err := template.New("body").Parse(*rt.TransformRule.RequestGenerationRule.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request body template: %w", err)
		}
		bodyBuffer := new(bytes.Buffer)
		if err := bodyTemplate.Execute(bodyBuffer, res); err != nil {
			return nil, fmt.Errorf("failed to execute request body template: %w", err)
		}
		bodyStr := bodyBuffer.String()
		body = &bodyStr
	}

	// Render the request path using path templates and resource definition
	pathTemplate, err := template.New("path").Parse(rt.TransformRule.RequestGenerationRule.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request path template: %w", err)
	}
	pathBuffer := new(bytes.Buffer)
	if err := pathTemplate.Execute(pathBuffer, res); err != nil {
		return nil, fmt.Errorf("failed to execute request path template: %w", err)
	}

	// Render the request header using header templates
	header, err := rt.generateHeaderOrQuery(rt.TransformRule.RequestGenerationRule.Header, res)
	if err != nil {
		return nil, fmt.Errorf("failed to generate request header: %w", err)
	}

	// Render the request query using query templates
	query, err := rt.generateHeaderOrQuery(rt.TransformRule.RequestGenerationRule.Query, res)
	if err != nil {
		return nil, fmt.Errorf("failed to generate request query: %w", err)
	}

	return &http.RequestHTTP{
		Request: rt.TransformRule.RequestGenerationRule.Request,
		Path:    pathBuffer.String(),
		Body:    body,
		Header:  header,
		Query:   query,
	}, nil
}

// generateHeaderOrQuery generates a header or query based on the provided templates and resource definition.
// It will loop through the templates and render them using the resource definition.
// It returns a map of header or query.
func (rt *RequestTransformerHTTP) generateHeaderOrQuery(templates map[string]utils.StringOrStringArray, vars resources.Resource,
) (map[string][]string, error) {
	result := make(map[string][]string)
	for key, tmplStrs := range templates {
		for _, tmplStr := range tmplStrs {
			tmpl, err := template.New(key).Parse(tmplStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s template: %w", key, err)
			}
			buffer := new(bytes.Buffer)
			if err := tmpl.Execute(buffer, vars); err != nil {
				return nil, fmt.Errorf("failed to execute %s template: %w", key, err)
			}
			if len(buffer.String()) != 0 {
				result[key] = append(result[key], buffer.String())
			}
		}
	}
	return result, nil
}

// ParseResponse parses the response from the provider and returns the state of the resource.
// It uses the DSL in the transform rule to parse the response.
func (rt *RequestTransformerHTTP) ParseResponse(response []byte) (state map[string]string, err error) {
	// Unmarshal the response into a map
	var responseMap map[string]interface{}
	if err := json.Unmarshal(response, &responseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Parse the response using the DSL
	state = make(map[string]string)
	for _, rule := range rt.TransformRule.ResponseParseRule {
		value, ok := getNestedValue(responseMap, rule.KeyPath)
		if ok {
			state[rule.KeyName] = value
		}
	}

	return state, nil
}

// getNestedValue gets the value of the nested key in the map.
func getNestedValue(m map[string]interface{}, valueIndex string) (string, bool) {
	keys := strings.Split(valueIndex, ".")
	for _, key := range keys {
		value, ok := m[key]
		if !ok {
			return "", false
		}
		if strValue, ok := value.(string); ok {
			return strValue, true
		}
		if nextMap, ok := value.(map[string]interface{}); ok {
			m = nextMap
		} else {
			return "", false
		}
	}
	return "", false
}
