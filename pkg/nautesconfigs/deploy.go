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

type DeployAppType string

const (
	DEPLOY_APP_TYPE_ARGOCD DeployAppType = "argocd"
)

type DeployApp struct {
	DefaultApp map[string]DeployAppType `yaml:"defaultApp"`
	ArgoCD     ArgoCD                   `yaml:"argocd"`
}

// Kustomize stores configurations of kustomize
type Kustomize struct {
	DefaultPath KustomizeDefaultPath `yaml:"defaultPath"`
}

// KustomizeDefaultPath stores default path of kustomize
type KustomizeDefaultPath struct {
	DefaultProject string `yaml:"defaultProject"`
}

// ArgoCD stores argocd configs
type ArgoCD struct {
	URL       string    `yaml:"url"`
	Namespace string    `yaml:"namespace"`
	Kustomize Kustomize `yaml:"kubestomize"`
}
