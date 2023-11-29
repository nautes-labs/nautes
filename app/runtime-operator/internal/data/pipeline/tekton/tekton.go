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

package tekton

import (
	"encoding/json"
	"fmt"
	"strconv"

	nautesv1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"

	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1alpha1"
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()
)

type tekton struct {
	currentRuntime string
	components     *component.ComponentList
	plgMgr         component.PipelinePluginManager
	opts           map[string]string
}

const (
	OptKeyDashBoardURL = "host"
)

const (
	pipelineType     = "tekton"
	taskNameGitClone = "git-clone"
)

const (
	ParamNameRevision         = "Revision"
	ParamNamePipelineRevision = "PipelineRevision"
	ParamNamePipelineFilePath = "PipelineFile"
)

const (
	WorkspaceSSHCreds      = "ssh-creds" //nolint:gosec
	WorkspacePipelineStore = "pipelines"
)

const (
	SecretNamePipelineReadOnlySSHKey = "pipeline-readonly-ssh-key"
)

type MountType string

const (
	MountTypeSecret MountType = "secret"
	MountTypePVC    MountType = "pvc"
	MountTypeEmpty  MountType = "empty"
)

var (
	ResourceTypeMapMountType = map[component.ResourceType]MountType{
		component.ResourceTypeCodeRepoSSHKey:      MountTypeSecret,
		component.ResourceTypeCodeRepoAccessToken: MountTypeSecret,
		component.ResourceTypeCAFile:              MountTypeSecret,
		component.ResourceTypeEmptyDir:            MountTypeEmpty,
	}
)

var (
	TektonFactory = &factory{}
)

type factory struct{}

func (tf *factory) NewStatus(_ []byte) (interface{}, error) {
	return nil, nil
}

func (tf *factory) NewComponent(opts nautesv1alpha1.Component, info *component.ComponentInitInfo, _ interface{}) (component.Pipeline, error) {
	if info.ClusterConnectInfo.ClusterKind != nautesv1alpha1.CLUSTER_KIND_KUBERNETES {
		return nil, fmt.Errorf("cluster type %s is not supported", info.ClusterConnectInfo.ClusterKind)
	}

	impl := &tekton{
		currentRuntime: info.RuntimeName,
		components:     info.Components,
		plgMgr:         info.PipelinePluginManager,
		opts:           map[string]string{},
	}

	if domain, ok := opts.Additions[OptKeyDashBoardURL]; ok {
		cluster, err := info.NautesResourceSnapshot.GetCluster(info.ClusterName)
		if err != nil {
			return nil, err
		}
		if len(cluster.Status.EntryPoints) != 0 {
			for _, entrypoint := range cluster.Status.EntryPoints {
				if entrypoint.HTTPSPort != 0 {
					impl.opts[OptKeyDashBoardURL] = fmt.Sprintf("https://%s:%d", domain, entrypoint.HTTPSPort)
				} else if entrypoint.HTTPPort != 0 {
					impl.opts[OptKeyDashBoardURL] = fmt.Sprintf("http://%s:%d", domain, entrypoint.HTTPPort)
				}
			}
		}
	}

	return impl, nil
}

func (t *tekton) CleanUp() error {
	return nil
}

func (t *tekton) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

// GetHooks will create hooks based on the user's input and event source type,
// and run the list of resources that need to be created in the environment for the hooks.
func (t *tekton) GetHooks(info component.HooksInitInfo) (*component.Hooks, []component.RequestResource, error) {
	pipelineRun := buildBaseHooks(info.BuiltinVars)
	runUserPipelineScript := buildRunPipelineScript(info.BuiltinVars, info.EventSourceType)
	addRunPipelineScriptIntoBasePipelineRun(pipelineRun, runUserPipelineScript)
	reqVars := buildBaseInputOverWrite(info, pipelineRun)
	resReqs := buildBaseResourceRequest(info.BuiltinVars)

	hooks, resReqs, err := t.createUserHook(info, *pipelineRun, reqVars, resReqs)
	if err != nil {
		return nil, nil, fmt.Errorf("create user hooks failed: %w", err)
	}

	return hooks, resReqs, nil
}

func (t *tekton) GetPipelineDashBoardURL() string {
	return t.opts[OptKeyDashBoardURL]
}

// createUserHook will append pre- and post-tasks to pipelineRun based on the hook information passed in by the user,
// and return a new PipelineRun with the input requirement list and resource requirement list.
func (t *tekton) createUserHook(
	info component.HooksInitInfo,
	pipelineRun v1alpha1.PipelineRun,
	requestVars []component.InputOverWrite,
	reqResources []component.RequestResource,
) (*component.Hooks, []component.RequestResource, error) {
	builtinVars := map[string]string{}
	for k, v := range info.BuiltinVars {
		builtinVars[string(k)] = v
	}

	for _, hook := range info.Hooks.PreHooks {
		hookBuildInfo := component.HookBuildData{
			UserVars:          hook.Vars,
			BuiltinVars:       builtinVars,
			EventSourceType:   string(info.EventSourceType),
			EventListenerType: info.EventListenerType,
		}
		task, err := t.convertHookToTask(hook, hookBuildInfo)
		if err != nil {
			return nil, nil, fmt.Errorf("convert hook to task failed: %w", err)
		}

		reqVars, err := addTaskIntoPipelineRun(&pipelineRun, *task, true)
		if err != nil {
			return nil, nil, fmt.Errorf("append task into init pipeline failed: %w", err)
		}

		requestVars = append(requestVars, reqVars...)
		reqResources = append(reqResources, task.requestResources...)
	}

	for _, hook := range info.Hooks.PostHooks {
		hookBuildInfo := component.HookBuildData{
			UserVars:          hook.Vars,
			BuiltinVars:       builtinVars,
			EventSourceType:   string(info.EventSourceType),
			EventListenerType: info.EventListenerType,
		}
		task, err := t.convertHookToTask(hook, hookBuildInfo)
		if err != nil {
			return nil, nil, fmt.Errorf("convert hook to task failed: %w", err)
		}

		reqVars, err := addTaskIntoPipelineRun(&pipelineRun, *task, false)
		if err != nil {
			return nil, nil, fmt.Errorf("append task into init pipeline failed: %w", err)
		}

		requestVars = append(requestVars, reqVars...)
		reqResources = append(reqResources, task.requestResources...)
	}

	return &component.Hooks{
			RequestVars: requestVars,
			Resource:    pipelineRun,
		},
		reqResources,
		nil
}

func addTaskIntoPipelineRun(pr *v1alpha1.PipelineRun, task Task, isPreTask bool) ([]component.InputOverWrite, error) { //nolint
	for i := range task.requestResources {
		appendResourceRequest(pr, task.requestResources[i])
	}

	var requestVars []component.InputOverWrite
	var taskIndex int
	if isPreTask {
		pipelineCloneIndex := getTaskIndexPipelineFileClone(pr.Spec.PipelineSpec.Tasks)
		taskIndex = pipelineCloneIndex - 1
		pr.Spec.PipelineSpec.Tasks[pipelineCloneIndex].RunAfter = []string{task.resource.Name}

		pr.Spec.PipelineSpec.Tasks = append(pr.Spec.PipelineSpec.Tasks[:pipelineCloneIndex],
			append([]v1alpha1.PipelineTask{task.resource},
				pr.Spec.PipelineSpec.Tasks[pipelineCloneIndex:]...,
			)...)
	} else {
		lastTaskName := getLastTaskName(pr.Spec.PipelineSpec.Tasks)
		task.resource.RunAfter = []string{lastTaskName}
		pr.Spec.PipelineSpec.Tasks = append(pr.Spec.PipelineSpec.Tasks, task.resource)
		taskIndex = len(pr.Spec.PipelineSpec.Tasks) - 1
	}

	for i, reqVar := range task.requestVars {
		requestVar, err := appendVar(pr, i, taskIndex, reqVar)
		if err != nil {
			return nil, err
		}
		requestVars = append(requestVars, *requestVar)
	}

	return requestVars, nil
}

func getTaskIndexPipelineFileClone(tasks []v1alpha1.PipelineTask) int {
	for i := range tasks {
		if tasks[i].Name == taskNameGitClone {
			return i
		}
	}
	return 0
}

func getLastTaskName(tasks []v1alpha1.PipelineTask) string {
	if len(tasks) == 0 {
		return ""
	}
	return tasks[len(tasks)-1].Name
}

// Task is to convert the string in pluginshared.Hook into a structure format
type Task struct {
	// RequestVars is the information that needs to be input from the outside for the hook to run.
	requestVars []component.PluginInputOverWrite
	// requestResources are the resources that need to be deployed in the environment for the hook to run properly.
	requestResources []component.RequestResource
	// resource is the code fragment that runs the hook, which is used to splice into the pre- and post-steps of the user pipeline.
	resource v1alpha1.PipelineTask
}

// convertHookToTask converts the user-passed hook into a Task object.
func (t *tekton) convertHookToTask(nautesHook nautesv1alpha1.Hook, info component.HookBuildData) (*Task, error) {
	hookFactory, err := t.plgMgr.GetHookFactory(pipelineType, nautesHook.Name)
	if err != nil {
		return nil, fmt.Errorf("get plugin failed: %w", err)
	}

	hook, err := hookFactory.BuildHook(nautesHook.Name, info)
	if err != nil {
		return nil, fmt.Errorf("build hook failed: %w", err)
	}

	pipelineTask := &v1alpha1.PipelineTask{}
	err = json.Unmarshal(hook.Resource, pipelineTask)
	if err != nil {
		return nil, fmt.Errorf("unmarshal pipeline task failed: %w", err)
	}

	if nautesHook.Alias != nil {
		pipelineTask.Name = *nautesHook.Alias
	}

	task := &Task{
		requestVars:      hook.RequestVars,
		requestResources: hook.RequestResources,
		resource:         *pipelineTask,
	}

	return task, nil
}

func appendVar(pr *v1alpha1.PipelineRun, paramIndex, taskIndex int, newVar component.PluginInputOverWrite) (*component.InputOverWrite, error) {
	var requestVar *component.InputOverWrite
	switch newVar.Source.Type {
	case component.SourceTypeEventSource:
		paramName := fmt.Sprintf("%s-%d", pr.Spec.PipelineSpec.Tasks[taskIndex].Name, paramIndex)
		hookName := pr.Spec.PipelineSpec.Tasks[taskIndex].Name

		pr.Spec.PipelineSpec.Params = append(pr.Spec.PipelineSpec.Params, v1beta1.ParamSpec{
			Name: paramName,
		})

		taskParamIndex, err := strconv.Atoi(newVar.Destination)
		if err != nil {
			return nil, fmt.Errorf("request var destination in hook %s format error", hookName)
		}
		if len(pr.Spec.PipelineSpec.Tasks[taskIndex].Params) <= taskParamIndex {
			return nil, fmt.Errorf("request var destination in hook %s not fount", hookName)
		}
		pr.Spec.PipelineSpec.Tasks[taskIndex].Params[taskParamIndex].Value = *v1beta1.NewArrayOrString(fmt.Sprintf("$(params.%s)", paramName))

		requestVar = &component.InputOverWrite{
			StaticeVar: newVar.Source.FromEventSource,
			Dest:       fmt.Sprintf("spec.params.%d.value", len(pr.Spec.Params)),
		}

		pr.Spec.Params = append(pr.Spec.Params, v1beta1.Param{
			Name:  paramName,
			Value: *v1beta1.NewArrayOrString(""),
		})
	default:
		return nil, fmt.Errorf("unknown source type %s", newVar.Source.Type)
	}

	return requestVar, nil
}

func appendResourceRequest(pr *v1alpha1.PipelineRun, req component.RequestResource) {
	switch ResourceTypeMapMountType[req.Type] {
	case MountTypeSecret:
		pr.Spec.Workspaces = append(pr.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name: req.ResourceName,
			Secret: &corev1.SecretVolumeSource{
				SecretName: req.ResourceName,
			},
		})
	case MountTypePVC:
		pr.Spec.Workspaces = append(pr.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name: req.ResourceName,
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: req.ResourceName,
				ReadOnly:  false,
			},
		})
	case MountTypeEmpty:
		pr.Spec.Workspaces = append(pr.Spec.Workspaces, v1beta1.WorkspaceBinding{
			Name:     req.ResourceName,
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		})
	}

	pr.Spec.PipelineSpec.Workspaces = append(pr.Spec.PipelineSpec.Workspaces, v1beta1.PipelineWorkspaceDeclaration{
		Name:     req.ResourceName,
		Optional: false,
	})
}

// buildBaseHooks will return a 'PipelineRun' with the tasks "pipeline file download" and "run pipeline file".
func buildBaseHooks(builtinVars map[component.BuiltinVar]string) *v1alpha1.PipelineRun {
	storageSize := "10M"
	pr := &v1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hooks-",
			Namespace:    builtinVars[component.VarNamespace],
		},
		Spec: v1alpha1.PipelineRunSpec{
			ServiceAccountName: builtinVars[component.VarServiceAccount],
			PipelineSpec: &v1alpha1.PipelineSpec{
				Tasks: []v1alpha1.PipelineTask{
					{
						Name: taskNameGitClone,
						TaskRef: &v1alpha1.TaskRef{
							Name: "git-clone",
							Kind: v1alpha1.ClusterTaskKind,
						},
						Params: []v1alpha1.Param{
							{
								Name:  "url",
								Value: *v1beta1.NewArrayOrString(builtinVars[component.VarPipelineCodeRepoURL]),
							},
							{
								Name:  "revision",
								Value: *v1beta1.NewArrayOrString(fmt.Sprintf("$(params.%s)", ParamNamePipelineRevision)),
							},
						},
						Workspaces: []v1alpha1.WorkspacePipelineTaskBinding{
							{
								Name:      "ssh-directory",
								Workspace: WorkspaceSSHCreds,
							},
							{
								Name:      "output",
								Workspace: WorkspacePipelineStore,
							},
						},
					},
					{
						Name: "pipeline-run",
						TaskRef: &v1alpha1.TaskRef{
							Name: "kubernetes-actions",
							Kind: v1alpha1.ClusterTaskKind,
						},
						RunAfter: []string{"git-clone"},
						Params: []v1alpha1.Param{
							{
								Name:  "image",
								Value: *v1beta1.NewArrayOrString("ghcr.io/nautes-labs/kubectl:v1.27.3-alpine-v3.18-v2"),
							},
							{
								Name: "script",
							},
						},
						Workspaces: []v1alpha1.WorkspacePipelineTaskBinding{
							{
								Name:      "manifest-dir",
								Workspace: WorkspacePipelineStore,
							},
						},
					},
				},
				Params: []v1alpha1.ParamSpec{
					{Name: ParamNameRevision},
					{Name: ParamNamePipelineRevision},
					{Name: ParamNamePipelineFilePath},
				},
				Workspaces: []v1beta1.WorkspacePipelineDeclaration{
					{Name: WorkspacePipelineStore},
					{Name: WorkspaceSSHCreds},
				},
			},
			Params: []v1beta1.Param{
				{
					Name:  ParamNameRevision,
					Value: *v1beta1.NewArrayOrString(""),
				},
				{
					Name:  ParamNamePipelineRevision,
					Value: *v1beta1.NewArrayOrString(""),
				},
				{
					Name:  ParamNamePipelineFilePath,
					Value: *v1beta1.NewArrayOrString(builtinVars[component.VarPipelineFilePath]),
				},
			},
			Workspaces: []v1alpha1.WorkspaceBinding{
				{
					Name: WorkspacePipelineStore,
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
							Resources: corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									corev1.ResourceStorage: resource.MustParse(storageSize),
								},
							},
						},
					},
				},
				{
					Name: WorkspaceSSHCreds,
					Secret: &corev1.SecretVolumeSource{
						SecretName: SecretNamePipelineReadOnlySSHKey,
					},
				},
			},
		},
	}

	return pr
}

// buildRunPipelineScript will build the script to run the pipeline file.
func buildRunPipelineScript(builtinVars map[component.BuiltinVar]string, eventType component.EventSourceType) string {
	var script string
	script += "SUFFIX=" + "`" + "openssl rand -hex 2" + "`"
	script += `
cat > ./kustomization.yaml << EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- $(params.PipelineFile)
nameSuffix: -n${SUFFIX}
`
	if pipelineLabel, ok := builtinVars[component.VarPipelineLabel]; ok {
		script += fmt.Sprintf(`
commonLabels:
  branch: %s
`, pipelineLabel)
	}

	if eventType == component.EventTypeGitlab {
		script += fmt.Sprintf(`
patches:
- patch: |-
    - op: replace
      path: /spec/params/0/value
      value: $(params.%s)
  target:
    kind: PipelineRun
`, ParamNamePipelineRevision)
	}
	script += fmt.Sprintf(`
EOF
cat ./kustomization.yaml
kubectl kustomize . | kubectl -n %s create -f -
`, builtinVars[component.VarNamespace])
	return script
}

// addRunPipelineScriptIntoBasePipelineRun will insert the run pipeline script into the pipeline run template.
func addRunPipelineScriptIntoBasePipelineRun(pr *v1alpha1.PipelineRun, script string) {
	pr.Spec.PipelineSpec.Tasks[1].Params[1].Value = *v1beta1.NewArrayOrString(script)
}

func buildBaseInputOverWrite(info component.HooksInitInfo, pipelineRun *v1alpha1.PipelineRun) []component.InputOverWrite {
	var reqVars []component.InputOverWrite
	if info.EventSourceType == component.EventTypeGitlab {
		reqVars = append(reqVars, component.InputOverWrite{
			BuiltinRequestVar: component.EventSourceVarRef,
			Dest:              "spec.params.0.value",
		})
	}

	if pipelineRevision, ok := info.BuiltinVars[component.VarPipelineRevision]; ok {
		pipelineRun.Spec.Params[1].Value.StringVal = pipelineRevision
	} else {
		reqVars = append(reqVars, component.InputOverWrite{
			BuiltinRequestVar: component.EventSourceVarRef,
			Dest:              "spec.params.1.value",
		})
	}
	return reqVars
}

func buildBaseResourceRequest(builtinVars map[component.BuiltinVar]string) []component.RequestResource {
	return []component.RequestResource{
		{
			Type:         component.ResourceTypeCodeRepoSSHKey,
			ResourceName: SecretNamePipelineReadOnlySSHKey,
			SSHKey: &component.ResourceRequestSSHKey{
				SecretInfo: component.SecretInfo{
					Type: component.SecretTypeCodeRepo,
					CodeRepo: &component.CodeRepo{
						ProviderType: builtinVars[component.VarCodeRepoProviderType],
						ID:           builtinVars[component.VarPipelineCodeRepoName],
						User:         "default",
						Permission:   component.CodeRepoPermissionReadOnly,
					},
				},
			},
		},
	}
}
