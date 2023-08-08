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
	_PilelineRuntime         = "Spec.PipelineRuntime"
	_DeploymentRuntime       = "Spec.DeploymentRuntime"
	_CodeRepo                = "Spec.CodeRepo"
	_Product                 = "Spec.Product"
	_ProjectsInProject       = "Spec.Projects.In"
	_PipelineTriggerPipeline = "Spec.PipelineTriggers.Pipeline"
	_Project                 = "Spec.Project"
	_PipelineSource          = "Spec.PipelineSource"
	_Destination             = "Spec.Destination"
	_ProjectsRef             = "Spec.ProjectsRef.In"
	_ManifestSource          = "Spec.ManifestSource.CodeRepo"
	_ClusterType             = "Spec.ClusterType"
	_Usage                   = "Spec.Usage"
	_WorkType                = "Spec.WorkerType"
)

const (
	thirdPartComponentsPath = "/opt/nautes/config/thirdPartComponents.yaml"
	EnvthirdPartComponents  = "thirdPartComponents"
)

type Component struct {
	Name        string   `yaml:"name"`
	Namespace   string   `yaml:"namespace"`
	Type        string   `yaml:"type"`
	Default     bool     `yaml:"default"`
	General     bool     `yaml:"general"`
	InstallPath []string `yaml:"installPath"`
}
