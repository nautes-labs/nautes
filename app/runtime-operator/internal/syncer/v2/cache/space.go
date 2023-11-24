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

package cache

import "github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"

type SpaceUsage map[string]utils.StringSet

func (su SpaceUsage) DeleteRuntime(runtimeName string) {
	delete(su, runtimeName)
}

func (su SpaceUsage) UpdateSpaceUsage(runtimeName string, spaces utils.StringSet) {
	su[runtimeName] = spaces
}

func (su SpaceUsage) ListSpaces(opts ...ListOption) utils.StringSet {
	listOpts := newListOptions(opts...)

	spaceSet := utils.NewStringSet()
	for runtimeName := range su {
		if listOpts.ExcludedRuntimeNames.Has(runtimeName) {
			continue
		}
		spaceSet.Set = spaceSet.Union(su[runtimeName].Set)
	}
	return spaceSet
}
