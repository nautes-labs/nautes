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
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/performer/middlewareruntime/resources"
)

// ConvertResourcesToAdvanceCallerResources converts custom resource declarations to resources recognized by AdvancedCaller.
func ConvertResourcesToAdvanceCallerResources(providerName, callerName, implementation, middlewareType string, res []resources.Resource) (callerRes string, opts map[string]interface{}, err error) {
	// 1. Get the template from the template directory based on providerName, callerName, and middlewareType.
	// 2. Render the template with the resource declarations.
	// 3. Return the template.
	return
}
