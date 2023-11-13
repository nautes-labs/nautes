package tekton

import (
	"context"
	"encoding/json"
	"fmt"

	nautesv1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
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
	components     *syncer.ComponentList
	status         *TektonStatus
}

type TektonStatus struct {
	Runtimes map[string]RuntimeResource `json:"runtimes,omitempty"`
}

type RuntimeResource struct {
	Space     syncer.Space                       `json:"space"`
	Resources map[ResourceType][]ResourceRequest `json:"resources,omitempty"`
}

type ResourceType string

const (
	ResourceTypeCodeRepoSSHKey ResourceType = "sshKey"
)

type ResourceRequest struct {
	Type   ResourceType           `json:"type"`
	SSHKey *ResourceRequestSSHKey `json:"sshKey,omitempty"`
}

type ResourceRequestSSHKey struct {
	ResourceName string            `json:"resourceName"`
	SecretInfo   syncer.SecretInfo `json:"secretInfo"`
}

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

func (tf *factory) NewComponent(_ nautesv1alpha1.Component, info *syncer.ComponentInitInfo, status interface{}) (syncer.Pipeline, error) {
	component := &tekton{
		components:     info.Components,
		currentRuntime: info.RuntimeName,
		status:         nil,
	}

	ttStatus, ok := status.(*TektonStatus)
	if !ok {
		return nil, fmt.Errorf("status is not a tekton status")
	}
	component.status = ttStatus

	return component, nil
}

func buildBaseHooks(buildInVars map[syncer.BuildInVar]string) *v1alpha1.PipelineRun {
	storageSize := "10M"
	pr := &v1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hooks-",
			Namespace:    buildInVars[syncer.VarNamespace],
		},
		Spec: v1alpha1.PipelineRunSpec{
			ServiceAccountName: buildInVars[syncer.VarServiceAccount],
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
								Value: *v1beta1.NewArrayOrString(buildInVars[syncer.VarPipelineCodeRepoURL]),
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
					Value: *v1beta1.NewArrayOrString(buildInVars[syncer.VarPipelineFilePath]),
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

func buildRunPipelineScript(buildInVars map[syncer.BuildInVar]string, eventType syncer.EventSourceType) string {
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
	if pipelineLabel, ok := buildInVars[syncer.VarPipelineLabel]; ok {
		script += fmt.Sprintf(`
commonLabels:
  nautes-pipeline-selector: %s
`, pipelineLabel)
	}

	if eventType == syncer.EventTypeGitlab {
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
`, buildInVars[syncer.VarNamespace])
	return script
}

func addRunPipelineScriptIntoBasePipelineRun(pr *v1alpha1.PipelineRun, script string) {
	pr.Spec.PipelineSpec.Tasks[1].Params[1].Value = *v1beta1.NewArrayOrString(script)
}

func buildBaseResourceRequest(buildInVars map[syncer.BuildInVar]string) []ResourceRequest {
	return []ResourceRequest{
		{
			Type: ResourceTypeCodeRepoSSHKey,
			SSHKey: &ResourceRequestSSHKey{
				ResourceName: SecretNamePipelineReadOnlySSHKey,
				SecretInfo: syncer.SecretInfo{
					Type: syncer.SecretTypeCodeRepo,
					CodeRepo: &syncer.CodeRepo{
						ProviderType: buildInVars[syncer.VarCodeRepoProviderType],
						ID:           buildInVars[syncer.VarPipelineCodeRepoName],
						User:         "default",
						Permission:   syncer.CodeRepoPermissionReadOnly,
					},
				},
			},
		},
	}
}

func (t *tekton) CleanUp() error {
	return nil
}

func (t *tekton) GetComponentMachineAccount() *syncer.MachineAccount {
	return nil
}

func (t *tekton) GetHooks(info syncer.HooksInitInfo) (syncer.Hooks, []interface{}, error) {
	pipelineRun := buildBaseHooks(info.BuildInVars)
	runUserPipelineScript := buildRunPipelineScript(info.BuildInVars, info.EventSourceType)
	addRunPipelineScriptIntoBasePipelineRun(pipelineRun, runUserPipelineScript)

	reqVars := []syncer.InputOverWrite{{
		Name: syncer.EventSourceVarRef,
		Dest: "spec.params.0.value",
	}}

	if pipelineRevision, ok := info.BuildInVars[syncer.VarPipelineRevision]; ok {
		pipelineRun.Spec.Params[1].Value.StringVal = pipelineRevision
	} else {
		reqVars = append(reqVars, syncer.InputOverWrite{
			Name: syncer.EventSourceVarRef,
			Dest: "spec.params.1.value",
		})
	}

	resReqs := buildBaseResourceRequest(info.BuildInVars)

	hooks := syncer.Hooks{
		RequestVars: reqVars,
		Resource:    pipelineRun,
	}
	return hooks, ConvertRequestsToInterfaceArray(resReqs), nil
}

func (t *tekton) CreateHookSpace(ctx context.Context, authInfo syncer.AuthInfo, space syncer.HookSpace) error {
	rawReqs, err := ConvertInterfaceToRequest(space.DeployResources)
	if err != nil {
		return err
	}
	reqs := classifyRequest(rawReqs)

	if runtimeStatus, ok := t.status.Runtimes[t.currentRuntime]; ok {
		if space.BaseSpace.Kubernetes.Namespace != runtimeStatus.Space.Kubernetes.Namespace {
			err := t.CleanUpHookSpace(ctx)
			if err != nil {
				return err
			}
		}
	}

	t.status.Runtimes[t.currentRuntime] = RuntimeResource{
		Space:     space.BaseSpace,
		Resources: reqs,
	}

	sshReqs := reqs[ResourceTypeCodeRepoSSHKey]
	for i := range sshReqs {
		err := t.CreateSSHKey(ctx, space.BaseSpace, authInfo, sshReqs[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tekton) CleanUpHookSpace(ctx context.Context) error {
	runtimeStatus, ok := t.status.Runtimes[t.currentRuntime]
	if !ok {
		return nil
	}

	sshReqs := runtimeStatus.Resources[ResourceTypeCodeRepoSSHKey]
	for i := range sshReqs {
		err := t.DeleteSSHKey(ctx, runtimeStatus.Space, sshReqs[i])
		if err != nil {
			return err
		}
	}

	delete(t.status.Runtimes, t.currentRuntime)
	return nil
}

func classifyRequest(reqs []ResourceRequest) map[ResourceType][]ResourceRequest {
	resMap := map[ResourceType][]ResourceRequest{}
	resMapHash := map[ResourceType]sets.Set[uint32]{}
	for i := range reqs {
		requestType := reqs[i].Type

		if _, ok := resMap[requestType]; !ok {
			resMap[requestType] = []ResourceRequest{}
			resMapHash[requestType] = sets.New[uint32]()
		}

		hash := utils.GetStructHash(reqs[i])
		if resMapHash[requestType].Has(hash) {
			continue
		}

		resMap[requestType] = append(resMap[requestType], reqs[i])
		resMapHash[requestType].Insert(hash)
	}
	return resMap
}

func (t *tekton) CreateSSHKey(ctx context.Context, space syncer.Space, authInfo syncer.AuthInfo, req ResourceRequest) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.CreateSecret(ctx, syncer.SecretRequest{
		Name:     sshReq.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: &authInfo,
		Destination: syncer.SecretRequestDestination{
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

func (t *tekton) DeleteSSHKey(ctx context.Context, space syncer.Space, req ResourceRequest) error {
	sshReq := req.SSHKey

	err := t.components.SecretSync.RemoveSecret(ctx, syncer.SecretRequest{
		Name:     sshReq.ResourceName,
		Source:   sshReq.SecretInfo,
		AuthInfo: nil,
		Destination: syncer.SecretRequestDestination{
			Name:  sshReq.ResourceName,
			Space: space,
		},
	})

	if err != nil {
		return fmt.Errorf("delete secret %s in space failed: %w", sshReq.ResourceName, err)
	}
	return nil
}

func ConvertRequestsToInterfaceArray(reqs []ResourceRequest) []interface{} {
	var interfaces []interface{}
	for i := range reqs {
		interfaces = append(interfaces, reqs[i])
	}
	return interfaces
}

func ConvertInterfaceToRequest(reqs []interface{}) ([]ResourceRequest, error) {
	resReqs := make([]ResourceRequest, len(reqs))
	for i := range reqs {
		req, ok := reqs[i].(ResourceRequest)
		if !ok {
			return nil, fmt.Errorf("convert interface to resource request failed")
		}
		resReqs[i] = req
	}
	return resReqs, nil
}
