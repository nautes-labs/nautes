/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"path/filepath"

	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline"
	corev1 "k8s.io/api/core/v1"
)

// WorkspaceDeclaration is a declaration of a volume that a Task requires.
type WorkspaceDeclaration struct {
	// Name is the name by which you can bind the volume at runtime.
	Name string `json:"name"`
	// Description is an optional human readable description of this volume.

	Description string `json:"description,omitempty"`
	// MountPath overrides the directory that the volume will be made available at.

	MountPath string `json:"mountPath,omitempty"`
	// ReadOnly dictates whether a mounted volume is writable. By default this
	// field is false and so mounted volumes are writable.
	ReadOnly bool `json:"readOnly,omitempty"`
	// Optional marks a Workspace as not being required in TaskRuns. By default
	// this field is false and so declared workspaces are required.
	Optional bool `json:"optional,omitempty"`
}

// GetMountPath returns the mountPath for w which is the MountPath if provided or the
// default if not.
func (w *WorkspaceDeclaration) GetMountPath() string {
	if w.MountPath != "" {
		return w.MountPath
	}
	return filepath.Join(pipeline.WorkspaceDir, w.Name)
}

// WorkspaceBinding maps a Task's declared workspace to a Volume.
type WorkspaceBinding struct {
	// Name is the name of the workspace populated by the volume.
	Name string `json:"name"`
	// SubPath is optionally a directory on the volume which should be used
	// for this binding (i.e. the volume will be mounted at this sub directory).

	SubPath string `json:"subPath,omitempty"`
	// VolumeClaimTemplate is a template for a claim that will be created in the same namespace.
	// The PipelineRun controller is responsible for creating a unique claim for each instance of PipelineRun.

	VolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
	// PersistentVolumeClaimVolumeSource represents a reference to a
	// PersistentVolumeClaim in the same namespace. Either this OR EmptyDir can be used.

	PersistentVolumeClaim *corev1.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// EmptyDir represents a temporary directory that shares a Task's lifetime.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// Either this OR PersistentVolumeClaim can be used.

	EmptyDir *corev1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// ConfigMap represents a configMap that should populate this workspace.

	ConfigMap *corev1.ConfigMapVolumeSource `json:"configMap,omitempty"`
	// Secret represents a secret that should populate this workspace.

	Secret *corev1.SecretVolumeSource `json:"secret,omitempty"`
}

// WorkspacePipelineDeclaration creates a named slot in a Pipeline that a PipelineRun
// is expected to populate with a workspace binding.
// Deprecated: use PipelineWorkspaceDeclaration type instead
type WorkspacePipelineDeclaration = PipelineWorkspaceDeclaration

// PipelineWorkspaceDeclaration creates a named slot in a Pipeline that a PipelineRun
// is expected to populate with a workspace binding.
type PipelineWorkspaceDeclaration struct {
	// Name is the name of a workspace to be provided by a PipelineRun.
	Name string `json:"name"`
	// Description is a human readable string describing how the workspace will be
	// used in the Pipeline. It can be useful to include a bit of detail about which
	// tasks are intended to have access to the data on the workspace.

	Description string `json:"description,omitempty"`
	// Optional marks a Workspace as not being required in PipelineRuns. By default
	// this field is false and so declared workspaces are required.
	Optional bool `json:"optional,omitempty"`
}

// WorkspacePipelineTaskBinding describes how a workspace passed into the pipeline should be
// mapped to a task's declared workspace.
type WorkspacePipelineTaskBinding struct {
	// Name is the name of the workspace as declared by the task
	Name string `json:"name"`
	// Workspace is the name of the workspace declared by the pipeline

	Workspace string `json:"workspace,omitempty"`
	// SubPath is optionally a directory on the volume which should be used
	// for this binding (i.e. the volume will be mounted at this sub directory).

	SubPath string `json:"subPath,omitempty"`
}

// WorkspaceUsage is used by a Step or Sidecar to declare that it wants isolated access
// to a Workspace defined in a Task.
type WorkspaceUsage struct {
	// Name is the name of the workspace this Step or Sidecar wants access to.
	Name string `json:"name"`
	// MountPath is the path that the workspace should be mounted to inside the Step or Sidecar,
	// overriding any MountPath specified in the Task's WorkspaceDeclaration.
	MountPath string `json:"mountPath"`
}
