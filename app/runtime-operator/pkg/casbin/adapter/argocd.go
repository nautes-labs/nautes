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

package rbac

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	_RBAC_FILE_NAME = "policy.csv"
)

var (
	roleTemplate = role{
		groupName: "",
		policies: []rolePolicy{
			{
				res: "projects",
				act: "get",
				obj: "%s",
			}, {
				res: "applications",
				act: "*",
				obj: "%s/*",
			}, {
				res: "exec",
				act: "*",
				obj: "%s/*",
			}, {
				res: "logs",
				act: "*",
				obj: "%s/*",
			},
		},
	}
	argocdModel, _ = model.NewModelFromString(`
[request_definition]
r = sub, res, act, obj

[policy_definition]
p = sub, res, act, obj, eft

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub) && globOrRegexMatch(r.res, p.res) && globOrRegexMatch(r.act, p.act) && globOrRegexMatch(r.obj, p.obj)
`)
)

type Adapter struct {
	K8sClient     client.Client
	rbacConfigMap *corev1.ConfigMap
	rawPolicies   string
	otherPolicies string
	// index by rolename
	roles map[string]role
}

type NewAdapterOption func(*Adapter)

func NewAdapter(opts ...NewAdapterOption) *Adapter {
	ada := &Adapter{
		roles: map[string]role{},
	}

	for _, fn := range opts {
		fn(ada)
	}

	return ada
}

func (ada *Adapter) GetClient() client.Client {
	return ada.K8sClient
}

func (ada *Adapter) LoadPolicyFromCluster(key types.NamespacedName) error {
	ada.rbacConfigMap = &corev1.ConfigMap{}
	err := ada.GetClient().Get(context.Background(), key, ada.rbacConfigMap)
	if err != nil {
		return fmt.Errorf("get config map failed: %w", err)
	}
	policies, ok := ada.rbacConfigMap.Data[_RBAC_FILE_NAME]
	if !ok {
		policies = ""
	}
	ada.rawPolicies = policies

	return ada.LoadPolicy(argocdModel)
}

func (ada *Adapter) SavePolicyToCluster(key types.NamespacedName) error {
	if ada.rbacConfigMap == nil {
		err := ada.GetClient().Get(context.Background(), key, ada.rbacConfigMap)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("get config map failed: %w", err)
			}

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Data: map[string]string{
					_RBAC_FILE_NAME: ada.ExportPolicies(),
				},
			}
			return ada.GetClient().Create(context.Background(), cm)
		}
	}
	if ada.rbacConfigMap.Data == nil {
		ada.rbacConfigMap.Data = make(map[string]string)
	}
	ada.rbacConfigMap.Data[_RBAC_FILE_NAME] = ada.ExportPolicies()

	return ada.GetClient().Update(context.Background(), ada.rbacConfigMap)
}

func (ada *Adapter) LoadPolicyFromString(policies string) error {
	ada.rawPolicies = policies
	return ada.LoadPolicy(argocdModel)
}

func (ada *Adapter) LoadPolicy(model model.Model) error {
	policies := strings.Split(ada.rawPolicies, "\n")
	for _, policy := range policies {
		if policy == "" || strings.HasPrefix(policy, "#") {
			return nil
		}

		reader := csv.NewReader(strings.NewReader(policy))
		reader.TrimLeadingSpace = true
		tokens, err := reader.Read()
		if err != nil {
			return err
		}

		if len(tokens) < 2 || len(tokens[0]) < 1 {
			return fmt.Errorf("invalid RBAC policy: %s", policy)
		}

		key := tokens[0]
		sec := key[:1]
		if _, ok := model[sec]; !ok {
			return fmt.Errorf("invalid RBAC policy: %s", policy)
		}
		if _, ok := model[sec][key]; !ok {
			return fmt.Errorf("invalid RBAC policy: %s", policy)
		}
		model[sec][key].Policy = append(model[sec][key].Policy, tokens[1:])

		switch sec {
		case "p":
			// if gourp info is not found after LoadPolicy, it will be a group without group name, and will not be used other actions
			ada.addPolicy(policy, tokens)
		case "g":
			ada.addGroup(policy, tokens)
		default:
			ada.otherPolicies = fmt.Sprintf("%s\n%s", ada.otherPolicies, policy)
		}

	}
	return nil
}

func (ada *Adapter) addPolicy(policy string, elements []string) {
	if strings.HasPrefix(elements[1], "role:") && len(elements) >= 5 {
		roleName := strings.SplitN(elements[1], ":", 2)[1]
		newPolicy := rolePolicy{
			res: elements[2],
			act: elements[3],
			obj: elements[4],
		}

		adaRole, ok := ada.roles[roleName]
		if !ok {
			ada.roles[roleName] = role{
				groupName: "",
				policies:  []rolePolicy{newPolicy},
			}
		} else {
			adaRole.policies = append(adaRole.policies, newPolicy)
			ada.roles[roleName] = adaRole
		}
	} else {
		ada.otherPolicies = fmt.Sprintf("%s\n%s", ada.otherPolicies, policy)
	}
}

func (ada *Adapter) addGroup(policy string, elements []string) {
	if strings.HasPrefix(elements[2], "role:") {
		roleName := strings.SplitN(elements[2], ":", 2)[1]
		adaRole, ok := ada.roles[roleName]
		if !ok {
			ada.roles[roleName] = role{
				groupName: elements[1],
				policies:  []rolePolicy{},
			}
		} else {
			adaRole.groupName = elements[1]
			ada.roles[roleName] = adaRole
		}
	} else {
		ada.otherPolicies = fmt.Sprintf("%s\n%s", ada.otherPolicies, policy)
	}
}

func (ada *Adapter) ExportPolicies() string {
	var policy string
	for roleName, role := range ada.roles {
		policy = fmt.Sprintf("%s\n%s", policy, role.ExportPolicies(roleName))
	}
	policy = fmt.Sprintf("%s\n%s", policy, ada.otherPolicies)

	return strings.TrimSpace(policy)
}

func (ada *Adapter) AddRole(groupName, roleName, projectName string) error {
	newRole := role{
		groupName: groupName,
		policies:  make([]rolePolicy, len(roleTemplate.policies)),
	}
	copy(newRole.policies, roleTemplate.policies)
	for i := range newRole.policies {
		newRole.policies[i].obj = fmt.Sprintf(newRole.policies[i].obj, projectName)
	}

	oldRole, ok := ada.roles[roleName]
	if ok && oldRole.ExportPolicies(roleName) == newRole.ExportPolicies(roleName) {
		return nil
	}

	ada.roles[roleName] = newRole
	return nil
}

func (ada *Adapter) DeleteRole(name string) error {
	delete(ada.roles, name)
	return nil
}

// SavePolicy saves all policy rules to the storage.
func (ada *Adapter) SavePolicy(model model.Model) error {
	return errors.New("not implemented")
}

// AddPolicy adds a policy rule to the storage.
// This is part of the Auto-Save feature.
func (ada *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemovePolicy removes a policy rule from the storage.
// This is part of the Auto-Save feature.
func (ada *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
// This is part of the Auto-Save feature.
func (ada *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}

// The modified version of LoadPolicyLine function defined in "persist" package of github.com/casbin/casbin.
// Uses CVS parser to correctly handle quotes in policy line.
func loadPolicyLine(line string, model model.Model) error {
	return errors.New("not implemented")
}

type role struct {
	groupName string
	policies  []rolePolicy
}

type rolePolicy struct {
	res string
	act string
	obj string
}

func (r *role) isLegal() bool {
	return r.groupName != "" && len(r.policies) != 0
}

// ExportPolicies get a group of policies for a role, if group name is not exist or no policies, it will return empty
func (r *role) ExportPolicies(roleName string) string {
	var policies string
	if !r.isLegal() {
		return policies
	}

	for _, policy := range r.policies {
		policyStr := fmt.Sprintf("p, role:%s, %s, %s, %s, allow", roleName, policy.res, policy.act, policy.obj)
		policies = fmt.Sprintf("%s\n%s", policies, policyStr)
	}

	groupPolicyStr := fmt.Sprintf("g, %s, role:%s\n", r.groupName, roleName)
	policies = fmt.Sprintf("%s\n%s", policies, groupPolicyStr)
	return strings.TrimSpace(policies)
}
