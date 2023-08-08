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

type EventBusType string

const (
	EventBusTypeArgoEvent EventBusType = "argo-events"
)

type EventBus struct {
	DefaultApp map[string]EventBusType `yaml:"defaultApp"`
	ArgoEvents ArgoEvents              `yaml:"argoEvents"`
}

type ArgoEvents struct {
	Namespace              string `yaml:"namespace"`
	TemplateServiceAccount string `yaml:"templateServiceAccount"`
}
