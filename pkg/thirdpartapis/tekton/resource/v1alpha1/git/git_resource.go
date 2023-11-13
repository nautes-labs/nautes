/*
Copyright 2019-2020 The Tekton Authors

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

package git

import (
	resource "github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/resource/v1alpha1"
)

var (
	gitSource = "git-source"
)

// Resource is an endpoint from which to get data which is required
// by a Build/Task for context (e.g. a repo from which to build an image).
type Resource struct {
	Name string                        `json:"name"`
	Type resource.PipelineResourceType `json:"type"`
	URL  string                        `json:"url"`
	// Git revision (branch, tag, commit SHA) to clone, and optionally the refspec to fetch from.
	// See https://git-scm.com/docs/gitrevisions#_specifying_revisions for more information.
	Revision   string `json:"revision"`
	Refspec    string `json:"refspec"`
	Submodules bool   `json:"submodules"`

	Depth      uint   `json:"depth"`
	SSLVerify  bool   `json:"sslVerify"`
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NOProxy    string `json:"noProxy"`
	GitImage   string `json:"-"`
}
