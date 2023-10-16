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

package v1alpha1

import "fmt"

type ProjectPipelineDestination struct {
	Environment string `json:"environment"`
	Namespace   string `json:"namespace"`
}

func (r *ProjectPipelineRuntime) GetProduct() string {
	return r.Namespace
}

func (r *ProjectPipelineRuntime) GetDestination() string {
	return r.Spec.Destination.Environment
}

func (r *ProjectPipelineRuntime) GetNamespaces() []string {
	namespace := r.Spec.Destination.Namespace
	if namespace == "" {
		namespace = r.Name
	}
	return []string{namespace}
}

func (r *ProjectPipelineRuntime) GetEventSource(name string) (*EventSource, error) {
	for _, eventSource := range r.Spec.EventSources {
		if eventSource.Name == name {
			return &eventSource, nil
		}
	}

	return nil, fmt.Errorf("can not find event source %s in runtime", name)
}

func (r *ProjectPipelineRuntime) GetPipeline(name string) (*Pipeline, error) {
	for _, pipeline := range r.Spec.Pipelines {
		if pipeline.Name == name {
			return &pipeline, nil
		}
	}

	return nil, fmt.Errorf("can not find pipeline %s in runtime", name)
}

func (r *ProjectPipelineRuntime) GetRuntimeType() RuntimeType {
	return RuntimeTypePipelineRuntime
}

func (r *ProjectPipelineRuntime) GetAccount() string {
	if r.Spec.Account == "" {
		return r.GetName()
	}
	return r.Spec.Account
}
