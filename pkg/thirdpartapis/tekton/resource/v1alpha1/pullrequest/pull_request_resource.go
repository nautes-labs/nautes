/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pullrequest

import (
	resourcev1alpha1 "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

const (
	prSource       = "pr-source"
	authTokenField = "authToken"
	// nolint: gosec
	authTokenEnv = "AUTH_TOKEN"
)

// Resource is an endpoint from which to get data which is required
// by a Build/Task for context.
type Resource struct {
	Name string                                `json:"name"`
	Type resourcev1alpha1.PipelineResourceType `json:"type"`

	// URL pointing to the pull request.
	// Example: https://github.com/owner/repo/pulls/1
	URL string `json:"url"`
	// SCM provider (github or gitlab today). This will be guessed from URL if not set.
	Provider string `json:"provider"`
	// Secrets holds a struct to indicate a field name and corresponding secret name to populate it.
	Secrets []resourcev1alpha1.SecretParam `json:"secrets"`

	PRImage                   string `json:"-"`
	InsecureSkipTLSVerify     bool   `json:"insecure-skip-tls-verify"`
	DisableStrictJSONComments bool   `json:"disable-strict-json-comments"`
}
