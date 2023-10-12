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

package service

// Filter field for list api.
const (
	FieldPipelineTriggersPipeline = "pipeline_triggers.pipeline"
	FeldManifestSourceCodeRepo    = "manifest_source.code_repo"
	FieldDestination              = "destination"
	FieldPipelineSource           = "pipeline_source"
	FieldProject                  = "project"
	FieldProjectsRef              = "projects_ref.in"
	FieldCodeRepo                 = "coderepo"
	FieldProduct                  = "product"
	FiledProjectsInProject        = "projects.in"
	FieldPilelineRuntime          = "pipeline_runtime"
	FieldDeploymentRuntime        = "deployment_runtime"
	FieldCluster                  = "cluster"
	FieldEnvType                  = "env_type"
	FieldClusterType              = "cluster_type"
	FieldUsage                    = "usage"
	FieldWorkType                 = "worker_type"
)

const (
	PilelineRuntime         = "Spec.PipelineRuntime"
	DeploymentRuntime       = "Spec.DeploymentRuntime"
	CodeRepo                = "Spec.CodeRepo"
	Product                 = "Spec.Product"
	ProjectsInProject       = "Spec.Projects.In"
	PipelineTriggerPipeline = "Spec.PipelineTriggers.Pipeline"
	Project                 = "Spec.Project"
	PipelineSource          = "Spec.PipelineSource"
	Destination             = "Spec.Destination"
	ProjectsRef             = "Spec.ProjectsRef.In"
	ManifestSource          = "Spec.ManifestSource.CodeRepo"
	ClusterType             = "Spec.ClusterType"
	Usage                   = "Spec.Usage"
	WorkType                = "Spec.WorkerType"
	RepoPrefix              = "repo-"
)
