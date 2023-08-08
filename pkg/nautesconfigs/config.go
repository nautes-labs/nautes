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

package configs

import (
	"gopkg.in/yaml.v2"
)

type Config struct {
	Deploy   DeployApp  `yaml:"deploy"`
	EventBus EventBus   `yaml:"eventbus"`
	Git      GitRepo    `yaml:"git"`
	Nautes   Nautes     `yaml:"nautes"`
	OAuth    OAuth      `yaml:"OAuth"`
	Pipeline Pipeline   `yaml:"pipeline"`
	Secret   SecretRepo `yaml:"secret"`
}

type GitType string

const (
	GIT_TYPE_GITLAB GitType = "gitlab"
	GIT_TYPE_GITHUB GitType = "github"
)

type GitRepo struct {
	Addr                 string  `yaml:"addr"`
	DefaultProductName   string  `yaml:"defaultProductName"`
	GitType              GitType `yaml:"gitType"`
	DefaultDeployKeyType string  `yaml:"defaultDeployKeyType"`
}

type Nautes struct {
	TenantName string `yaml:"tenantName"`
	Namespace  string `yaml:"namespace"`
	// The git repo where store runtime template
	RuntimeTemplateSource string            `yaml:"runtimeTemplateSource"`
	ServiceAccount        map[string]string `yaml:"serviceAccount"`
}

func NewConfig(cfgString string) (*Config, error) {
	cfg := &Config{
		Nautes: Nautes{
			TenantName:            "",
			Namespace:             "nautes",
			RuntimeTemplateSource: "",
			ServiceAccount: map[string]string{
				"Api":     "api-server",
				"Argo":    "argo-controller-server",
				"Base":    "base-controller-server",
				"Cluster": "cluster-controller-server",
				"Repo":    "repo-controller-server",
				"Runtime": "runtime-controller-server",
			},
		},
		EventBus: EventBus{
			DefaultApp: map[string]EventBusType{
				"kubernetes": EventBusTypeArgoEvent,
			},
			ArgoEvents: ArgoEvents{
				Namespace:              "argo-events",
				TemplateServiceAccount: "argo-events-sa",
			},
		},
		OAuth: OAuth{
			OAuthType: OAuthTypeDex,
		},
		Git: GitRepo{
			Addr:                 "https://www.github.com",
			DefaultProductName:   "default.project",
			GitType:              GIT_TYPE_GITHUB,
			DefaultDeployKeyType: "ecdsa",
		},
		Deploy: DeployApp{
			DefaultApp: map[string]DeployAppType{
				"kubernetes": DEPLOY_APP_TYPE_ARGOCD,
			},
			ArgoCD: ArgoCD{
				URL:       "https://argocd-server.argocd.svc",
				Namespace: "argocd",
				Kustomize: Kustomize{
					DefaultPath: KustomizeDefaultPath{
						DefaultProject: "production",
					},
				},
			},
		},
		Pipeline: Pipeline{
			DefaultApp: map[string]PipelineType{
				"kubernetes": PipelineTypeTekton,
			},
		},
		Secret: SecretRepo{
			RepoType: SECRET_STORE_VAULT,
			OperatorName: map[string]string{
				"Api":     "API",
				"Argo":    "ARGO",
				"Base":    "BASE",
				"Cluster": "CLUSTER",
				"Repo":    "REPO",
				"Runtime": "RUNTIME",
			},
			Vault: Vault{
				Addr:           "http://127.0.0.1:8200",
				ProxyAddr:      "http://127.0.0.1:8000",
				MountPath:      "",
				CABundle:       "",
				PKIPath:        "./ca/",
				Token:          "",
				Namesapce:      "vault",
				ServiceAccount: "vault-auth",
			},
		},
	}

	err := yaml.Unmarshal([]byte(cfgString), cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Secret.Vault.MountPath == "" {
		cfg.Secret.Vault.MountPath = cfg.Nautes.TenantName
	}

	return cfg, nil
}
