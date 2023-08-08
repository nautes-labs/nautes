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

package validate

import (
	"context"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
)

type ValidateClient interface {
	GetCodeRepo(ctx context.Context, repoName string) (*resourcev1alpha1.CodeRepo, error)
	ListCodeRepoBinding(ctx context.Context, productName, repoName string) (*resourcev1alpha1.CodeRepoBindingList, error)
	ListDeploymentRuntime(ctx context.Context, productName string) (*resourcev1alpha1.DeploymentRuntimeList, error)
	ListProjectPipelineRuntime(ctx context.Context, productName string) (*resourcev1alpha1.ProjectPipelineRuntimeList, error)
}
