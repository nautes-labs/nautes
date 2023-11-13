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

import (
	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	component "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

type GetRuntimeResourceFunction func(runtime v1alpha1.Runtime) (RuntimeResource, error)

type AccountUsage struct {
	Accounts map[string]AccountResource `yaml:"accounts"`
}

func (au *AccountUsage) AddRuntime(accountName, runtimeName string) {
	if au.Accounts == nil {
		au.Accounts = map[string]AccountResource{}
	}

	_, ok := au.Accounts[accountName]
	if !ok {
		au.Accounts[accountName] = AccountResource{
			Runtimes: map[string]RuntimeResource{},
		}
	}

	_, ok = au.Accounts[accountName].Runtimes[runtimeName]
	if ok {
		return
	}

	au.Accounts[accountName].Runtimes[runtimeName] = RuntimeResource{
		Spaces:      utils.StringSet{},
		Permissions: []component.SecretInfo{},
	}
}

func (au *AccountUsage) DeleteRuntime(accountName, runtimeName string) {
	if au.Accounts == nil {
		return
	}
	_, ok := au.Accounts[accountName]
	if !ok {
		return
	}
	delete(au.Accounts[accountName].Runtimes, runtimeName)

	if len(au.Accounts[accountName].Runtimes) == 0 {
		delete(au.Accounts, accountName)
	}
}

func (au *AccountUsage) UpdateSpaces(accountName, runtimeName string, spaces utils.StringSet) {
	au.AddRuntime(accountName, runtimeName)
	runtimeUsage := au.Accounts[accountName].Runtimes[runtimeName]
	runtimeUsage.Spaces = spaces
	au.Accounts[accountName].Runtimes[runtimeName] = runtimeUsage
}

func (au *AccountUsage) UpdatePermissions(accountName, runtimeName string, permissions []component.SecretInfo) {
	au.AddRuntime(accountName, runtimeName)
	runtimeUsage := au.Accounts[accountName].Runtimes[runtimeName]
	runtimeUsage.Permissions = permissions
	au.Accounts[accountName].Runtimes[runtimeName] = runtimeUsage
}

type AccountResource struct {
	Runtimes map[string]RuntimeResource `yaml:"runtimes"`
}

type RuntimeResource struct {
	Spaces      utils.StringSet        `yaml:"spaces,omitempty"`
	Permissions []component.SecretInfo `yaml:"permissions,omitempty"`
}

type listOptions struct {
	ExcludedRuntimeNames utils.StringSet
}

func newListOptions(opts ...ListOption) listOptions {
	listOpts := &listOptions{
		ExcludedRuntimeNames: utils.NewStringSet(),
	}

	for _, fn := range opts {
		fn(listOpts)
	}

	return *listOpts
}

type ListOption func(*listOptions)

func ExcludedRuntimes(runtimeNames []string) ListOption {
	return func(lopt *listOptions) {
		lopt.ExcludedRuntimeNames = utils.NewStringSet(runtimeNames...)
	}
}

func (ar AccountResource) ListAccountSpaces(opts ...ListOption) utils.StringSet {
	listOpts := newListOptions(opts...)

	spaceSet := utils.NewStringSet()
	for name, runtime := range ar.Runtimes {
		if listOpts.ExcludedRuntimeNames.Has(name) {
			continue
		}
		spaceSet.Set = spaceSet.Union(runtime.Spaces.Set)
	}
	return spaceSet
}

func (ar AccountResource) ListPermissions(opts ...ListOption) []component.SecretInfo {
	listOpts := newListOptions(opts...)

	var permissions []component.SecretInfo
	hashSet := sets.New[uint32]()
	for name := range ar.Runtimes {
		if listOpts.ExcludedRuntimeNames.Has(name) {
			continue
		}

		for i, permission := range ar.Runtimes[name].Permissions {
			hash := utils.GetStructHash(permission)
			if hashSet.Has(hash) {
				continue
			}

			hashSet.Insert(hash)
			permissions = append(permissions, ar.Runtimes[name].Permissions[i])
		}
	}

	return permissions
}
