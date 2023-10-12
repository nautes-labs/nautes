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

package biz

import (
	"fmt"

	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

const (
	_GitUser  = "gittestsuer"
	_GitEmail = "gittestsuer@nautes.com"
)

var (
	nautesConfigs = &nautesconfigs.Config{
		Nautes: nautesconfigs.Nautes{
			TenantName:            "",
			Namespace:             "nautes",
			RuntimeTemplateSource: "https://gitlab.test.io/nautes-labs/cluster-templates.git",
			ServiceAccount: map[string]string{
				"Api":     "api-server",
				"Argo":    "argo-operator",
				"Base":    "base-operator",
				"Cluster": "cluster-operator",
				"Repo":    "repo-operator",
				"Runtime": "runtime-operator",
			},
		},
		Git: nautesconfigs.GitRepo{
			Addr:                 "https://gitlab.xxx.io",
			DefaultProductName:   "default.project",
			GitType:              "gitlab",
			DefaultDeployKeyType: "rsa",
		},
		Deploy: nautesconfigs.DeployApp{
			ArgoCD: nautesconfigs.ArgoCD{
				Namespace: "argocd",
				Kustomize: nautesconfigs.Kustomize{
					DefaultPath: nautesconfigs.KustomizeDefaultPath{
						DefaultProject: "production",
					},
				},
			},
		},
		Secret: nautesconfigs.SecretRepo{
			RepoType: "vault",
			Vault: nautesconfigs.Vault{
				Addr:           "https://vault.xxxx.io:8200",
				ProxyAddr:      "https://vault.proxy.io:8000",
				CABundle:       "ca bundle",
				PKIPath:        "PKIPath",
				MountPath:      "cluster-server",
				Token:          "token",
				Namesapce:      "vault",
				ServiceAccount: "vault-auth",
			},
			OperatorName: map[string]string{
				"Api":     "API",
				"Argo":    "ARGO",
				"Base":    "BASE",
				"Cluster": "CLUSTER",
				"Repo":    "REPO",
				"Runtime": "RUNTIME",
			},
		},
	}
	// git platform group
	defaultGroupName    = "API_SERVER_TEST_GROUP"
	defaultProductGroup = &Group{
		ID:       int32(560),
		Name:     defaultGroupName,
		Path:     defaultGroupName,
		WebUrl:   "https://github.com/groups/" + defaultGroupName,
		ParentId: int32(0),
	}
	// git platform default project
	defaultProjectName = nautesConfigs.Git.DefaultProductName
	defautlProject     = &Project{
		ID:                int32(297),
		Name:              defaultProjectName,
		Path:              defaultProjectName,
		WebUrl:            fmt.Sprintf("https://github.com/test-2/%s", defaultProjectName),
		SshUrlToRepo:      fmt.Sprintf("ssh://git@github.com:2222/test-2/%s.git", defaultProjectName),
		HttpUrlToRepo:     fmt.Sprintf("https://github.com:2222/test-2/%s.git", defaultProjectName),
		PathWithNamespace: fmt.Sprintf("%s/%s", defaultProductGroup.Path, defaultProjectName),
		Namespace: &ProjectNamespace{
			ID:   3123,
			Name: defaultGroupName,
			Path: defaultGroupName,
		},
	}
	defaultProjectPath = fmt.Sprintf("%v/%v", defaultProductGroup.Path, defaultProjectName)
	defaultProductId   = fmt.Sprintf("%s%d", ProductPrefix, int(defaultProductGroup.ID))
	// node
	emptyNodes = nodestree.Node{
		Name:     defaultProjectName,
		Path:     defaultProjectName,
		IsDir:    true,
		Level:    1,
		Children: []*nodestree.Node{},
	}
	localRepositoryPath = fmt.Sprintf("/tmp/defaultProjectDir/%s", defaultProjectName)
)
