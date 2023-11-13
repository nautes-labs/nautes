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
