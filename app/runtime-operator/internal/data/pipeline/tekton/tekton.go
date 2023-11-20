package tekton

import (
	"context"
	"encoding/json"
	"fmt"

	nautesv1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	pluginmanager "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/manager"
	shared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/tekton"

	pluginshared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/shared"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1alpha1"
	"github.com/nautes-labs/nautes/pkg/thirdpartapis/tekton/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type tekton struct {
	currentRuntime string
	components     *component.ComponentList
	status         *TektonStatus
	plgMgr         pluginmanager.PipelinePluginManager
}

type TektonStatus struct {
	Runtimes map[string]RuntimeResource `json:"runtimes,omitempty"`
}

type RuntimeResource struct {
	Space     component.Space          `json:"space"`
	Resources []shared.ResourceRequest `json:"resources,omitempty"`
}

const pipelineType = "tekton"

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

var (
	TektonFactory = &factory{}
)

type factory struct{}

func (tf *factory) NewStatus(rawStatus []byte) (interface{}, error) {
	status := &TektonStatus{
		Runtimes: map[string]RuntimeResource{},
	}

	if len(rawStatus) == 0 {
		return status, nil
	}

	err := json.Unmarshal(rawStatus, status)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status")
	}

	return status, nil
}

func (tf *factory) NewComponent(_ nautesv1alpha1.Component, info *component.ComponentInitInfo, status interface{}) (component.Pipeline, error) {
	impl := &tekton{
		currentRuntime: info.RuntimeName,
		components:     info.Components,
		status:         nil,
		plgMgr:         info.PipelinePluginManager,
	}

	ttStatus, ok := status.(*TektonStatus)
	if !ok {
		return nil, fmt.Errorf("status is not a tekton status")
	}
	impl.status = ttStatus

	return impl, nil
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
						Name: "git-clone",
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
  nautes-pipeline-selector: %s
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

func buildBaseResourceRequest(builtinVars map[component.BuiltinVar]string) []shared.ResourceRequest {
	return []shared.ResourceRequest{
		{
			Type: shared.ResourceTypeCodeRepoSSHKey,
			SSHKey: &shared.ResourceRequestSSHKey{
				ResourceName: SecretNamePipelineReadOnlySSHKey,
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

func (t *tekton) CleanUp() error {
	return nil
}

func (t *tekton) GetComponentMachineAccount() *component.MachineAccount {
	return nil
}

// GetHooks will create hooks based on the user's input and event source type,
// and run the list of resources that need to be created in the environment for the hooks.
func (t *tekton) GetHooks(info component.HooksInitInfo) (*component.Hooks, []interface{}, error) {
	pipelineRun := buildBaseHooks(info.BuiltinVars)
	runUserPipelineScript := buildRunPipelineScript(info.BuiltinVars, info.EventSourceType)
	addRunPipelineScriptIntoBasePipelineRun(pipelineRun, runUserPipelineScript)
	reqVars := buildBaseInputOverWrite(info, pipelineRun)
	resReqs := buildBaseResourceRequest(info.BuiltinVars)

	hooks, resReqs, err := t.CreateUserHook(info, *pipelineRun, reqVars, resReqs)
	if err != nil {
		return nil, nil, fmt.Errorf("create user hooks failed: %w", err)
	}

	return hooks, ConvertRequestsToInterfaceArray(resReqs), nil
}

// CreateUserHook will append pre- and post-tasks to pipelineRun based on the hook information passed in by the user,
// and return a new PipelineRun with the input requirement list and resource requirement list.
func (t *tekton) CreateUserHook(
	info component.HooksInitInfo,
	pipelineRun v1alpha1.PipelineRun,
	requestVars []component.InputOverWrite,
	reqResources []shared.ResourceRequest,
) (*component.Hooks, []shared.ResourceRequest, error) {
	builtinVars := map[string]string{}
	for k, v := range info.BuiltinVars {
		builtinVars[string(k)] = v
	}

	preTasks, _, reqRes, err := t.ConvertHooksToPipelineResource(builtinVars, info.EventSourceType, info.EventListenerType, info.Hooks.PreHooks)
	if err != nil {
		return nil, nil, err
	}
	reqResources = append(reqResources, reqRes...)

	if len(preTasks) != 0 {
		pipelineRun.Spec.PipelineSpec.Tasks = append(preTasks, pipelineRun.Spec.PipelineSpec.Tasks...)
		pipelineRun.Spec.PipelineSpec.Tasks[len(preTasks)].RunAfter = []string{getLastTaskName(preTasks)}
	}

	postTasks, _, reqRes, err := t.ConvertHooksToPipelineResource(builtinVars, info.EventSourceType, info.EventListenerType, info.Hooks.PostHooks)
	if err != nil {
		return nil, nil, err
	}
	reqResources = append(reqResources, reqRes...)

	if len(postTasks) != 0 {
		postTasks[0].RunAfter = []string{getLastTaskName(pipelineRun.Spec.PipelineSpec.Tasks)}
		pipelineRun.Spec.PipelineSpec.Tasks = append(pipelineRun.Spec.PipelineSpec.Tasks, postTasks...)
	}

	return &component.Hooks{
			RequestVars: requestVars,
			Resource:    pipelineRun,
		},
		reqResources,
		nil
}

func (t *tekton) ConvertHooksToPipelineResource(builtinVars map[string]string, eventSourceType component.EventSourceType, eventListenerType string, hooks []nautesv1alpha1.Hook) (
	tasks []v1alpha1.PipelineTask,
	reqInputs []pluginshared.InputOverWrite,
	reqResources []shared.ResourceRequest,
	err error) {
	for i, hook := range hooks {
		hookBuildInfo := pluginshared.HookBuildData{
			UserVars:          hook.Vars,
			BuiltinVars:       builtinVars,
			EventSourceType:   string(eventSourceType),
			EventListenerType: eventListenerType,
		}
		task, err := t.convertHookToTask(hook, hookBuildInfo)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("convert hook to task failed: %w", err)
		}

		reqResources = append(reqResources, task.requestResources...)

		if i == 0 {
			tasks = append(tasks, task.resource)
			continue
		}

		lastTaskName := tasks[len(tasks)-1].Name
		task.resource.RunAfter = []string{lastTaskName}
		tasks = append(tasks, task.resource)
	}

	return
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
	requestVars []pluginshared.InputOverWrite
	// requestResources are the resources that need to be deployed in the environment for the hook to run properly.
	requestResources []shared.ResourceRequest
	// resource is the code fragment that runs the hook, which is used to splice into the pre- and post-steps of the user pipeline.
	resource v1alpha1.PipelineTask
}

// convertHookToTask 把用户传入的 hook 转换成 Task 对象。
func (t *tekton) convertHookToTask(nautesHook nautesv1alpha1.Hook, info pluginshared.HookBuildData) (*Task, error) {
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

	var reqResources []shared.ResourceRequest
	for i := range hook.RequestResources {
		res := &shared.ResourceRequest{}
		err = json.Unmarshal(hook.RequestResources[i], res)
		if err != nil {
			return nil, fmt.Errorf("unmarshal request resource failed: %w", err)
		}
		reqResources = append(reqResources, *res)
	}

	task := &Task{
		requestVars:      hook.RequestVars,
		requestResources: reqResources,
		resource:         *pipelineTask,
	}

	return task, nil
}

// CreateHookSpace will create resources in the corresponding space based on the incoming resource request list.
// If the resource needs to access the secret management system, use authInfo for authentication.
//
// Clean up logic:
// - If the runtime space changes, all resources on the old space will be cleared.
// - If the requested resource changes, the resources that are no longer used will be cleared.
func (t *tekton) CreateHookSpace(ctx context.Context, authInfo component.AuthInfo, space component.HookSpace) error {
	rawReqs, err := ConvertInterfaceToRequest(space.DeployResources)
	if err != nil {
		return err
	}
	resourceRequests := RemoveDuplicateRequests(rawReqs)

	if runtimeStatus, ok := t.status.Runtimes[t.currentRuntime]; ok {
		if space.BaseSpace.Kubernetes.Namespace != runtimeStatus.Space.Kubernetes.Namespace {
			err := t.CleanUpHookSpace(ctx)
			if err != nil {
				return err
			}
		} else if removeRes := GetUnusedResources(resourceRequests, runtimeStatus.Resources); len(removeRes) != 0 {
			for i := range removeRes {
				err := t.DeleteResource(ctx, space.BaseSpace, removeRes[i])
				if err != nil {
					return fmt.Errorf("remove resource failed: %w", err)
				}
			}
		}
	}

	t.status.Runtimes[t.currentRuntime] = RuntimeResource{
		Space:     space.BaseSpace,
		Resources: resourceRequests,
	}

	for i := range resourceRequests {
		err := t.CreateResource(ctx, space.BaseSpace, authInfo, resourceRequests[i])
		if err != nil {
			return fmt.Errorf("create resource failed: %w", err)
		}
	}

	return nil
}

// CleanUpHookSpace will clear all the resources required for the runtime deployed on the space based on the currently processed runtime.
func (t *tekton) CleanUpHookSpace(ctx context.Context) error {
	runtimeStatus, ok := t.status.Runtimes[t.currentRuntime]
	if !ok {
		return nil
	}

	for i := range runtimeStatus.Resources {
		err := t.DeleteResource(ctx, runtimeStatus.Space, runtimeStatus.Resources[i])
		if err != nil {
			return err
		}
	}

	delete(t.status.Runtimes, t.currentRuntime)
	return nil
}

// RemoveDuplicateRequests will remove duplicate resource requests in reqs.
func RemoveDuplicateRequests(reqs []shared.ResourceRequest) []shared.ResourceRequest {
	resMap := []shared.ResourceRequest{}
	resMapHash := sets.New[uint32]()
	for i := range reqs {
		hash := utils.GetStructHash(reqs[i])
		if resMapHash.Has(hash) {
			continue
		}

		resMap = append(resMap, reqs[i])
		resMapHash.Insert(hash)
	}
	return resMap
}

// GetUnusedResources compares newRes and oldRes to find 'ResourceRequest' that are only in oldRes.
func GetUnusedResources(newRes []shared.ResourceRequest, oldRes []shared.ResourceRequest) []shared.ResourceRequest {
	var removeRes []shared.ResourceRequest
	newHash := sets.New[uint32]()
	for i := range newRes {
		newHash.Insert(utils.GetStructHash(newRes[i]))
	}
	for i := range oldRes {
		if newHash.Has(utils.GetStructHash(oldRes[i])) {
			continue
		}
		removeRes = append(removeRes, oldRes[i])
	}
	return removeRes
}

func (t *tekton) CreateResource(ctx context.Context, space component.Space, authInfo component.AuthInfo, req shared.ResourceRequest) error {
	switch req.Type {
	case shared.ResourceTypeCodeRepoSSHKey:
		return t.CreateSSHKey(ctx, space, authInfo, req)
	case shared.ResourceTypeCodeRepoAccessToken:
		return t.CreateAccessToken(ctx, space, authInfo, req)
	default:
		return fmt.Errorf("unknown request type %s", req.Type)
	}
}

func (t *tekton) DeleteResource(ctx context.Context, space component.Space, req shared.ResourceRequest) error {
	switch req.Type {
	case shared.ResourceTypeCodeRepoSSHKey:
		return t.DeleteSSHKey(ctx, space, req)
	case shared.ResourceTypeCodeRepoAccessToken:
		return t.DeleteAccessToken(ctx, space, req)
	default:
		return fmt.Errorf("unknown request type %s", req.Type)
	}
}

func (t *tekton) CreateSSHKey(ctx context.Context, space component.Space, authInfo component.AuthInfo, req shared.ResourceRequest) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.CreateSecret(ctx, component.SecretRequest{
		Name:     sshReq.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: &authInfo,
		Destination: component.SecretRequestDestination{
			Name:   sshReq.ResourceName,
			Space:  space,
			Format: `id_rsa: '{{ .deploykey | toString }}'`,
		},
	})
	if err != nil {
		return fmt.Errorf("create secret %s in space failed: %w", sshReq.ResourceName, err)
	}

	return nil
}

func (t *tekton) DeleteSSHKey(ctx context.Context, space component.Space, req shared.ResourceRequest) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.RemoveSecret(ctx, component.SecretRequest{
		Name:     sshReq.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: nil,
		Destination: component.SecretRequestDestination{
			Name:  sshReq.ResourceName,
			Space: space,
		},
	})

	if err != nil {
		return fmt.Errorf("delete secret %s in space failed: %w", sshReq.ResourceName, err)
	}
	return nil
}

func (t *tekton) CreateAccessToken(ctx context.Context, space component.Space, authInfo component.AuthInfo, req shared.ResourceRequest) error {
	tokenReq := req.AccessToken

	err := t.components.SecretSync.CreateSecret(ctx, component.SecretRequest{
		Name:     tokenReq.ResourceName,
		Source:   tokenReq.SecretInfo,
		AuthInfo: &authInfo,
		Destination: component.SecretRequestDestination{
			Name:   tokenReq.ResourceName,
			Space:  space,
			Format: `id_rsa: '{{ .accesstoken | toString }}'`,
		},
	})
	if err != nil {
		return fmt.Errorf("create secret %s in space failed: %w", tokenReq.ResourceName, err)
	}

	return nil
}

func (t *tekton) DeleteAccessToken(ctx context.Context, space component.Space, req shared.ResourceRequest) error {
	tokenReq := req.AccessToken

	err := t.components.SecretSync.RemoveSecret(ctx, component.SecretRequest{
		Name:     tokenReq.ResourceName,
		Source:   tokenReq.SecretInfo,
		AuthInfo: nil,
		Destination: component.SecretRequestDestination{
			Name:  tokenReq.ResourceName,
			Space: space,
		},
	})

	if err != nil {
		return fmt.Errorf("delete secret %s in space failed: %w", tokenReq.ResourceName, err)
	}
	return nil
}

func ConvertRequestsToInterfaceArray(reqs []shared.ResourceRequest) []interface{} {
	var interfaces []interface{}
	for i := range reqs {
		interfaces = append(interfaces, reqs[i])
	}
	return interfaces
}

func ConvertInterfaceToRequest(reqs []interface{}) ([]shared.ResourceRequest, error) {
	resReqs := make([]shared.ResourceRequest, len(reqs))
	for i := range reqs {
		req, ok := reqs[i].(shared.ResourceRequest)
		if !ok {
			return nil, fmt.Errorf("convert interface to resource request failed")
		}
		resReqs[i] = req
	}
	return resReqs, nil
}
