package cache

import (
	"fmt"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/utils"
)

type GetRuntimeResourceFunction func(runtime v1alpha1.Runtime) (RuntimeResource, error)

var (
	GetRuntimeResourceFunctionMap = map[v1alpha1.RuntimeType]GetRuntimeResourceFunction{
		v1alpha1.RuntimeTypeDeploymentRuntime: GetRuntimeResourceFromDeploymentRuntime,
		v1alpha1.RuntimeTypePipelineRuntime:   GetRuntimeResourceFromPipelineRuntime,
	}
)

func GetRuntimeResourceFromDeploymentRuntime(runtime v1alpha1.Runtime) (RuntimeResource, error) {
	deploymentRuntime, ok := runtime.(*v1alpha1.DeploymentRuntime)
	if !ok {
		return RuntimeResource{}, fmt.Errorf("runtime is not a deployment runtime")
	}

	return RuntimeResource{
		Spaces:    utils.NewStringSet(deploymentRuntime.GetNamespaces()...),
		CodeRepos: utils.NewStringSet(deploymentRuntime.Spec.ManifestSource.CodeRepo),
	}, nil
}

func GetRuntimeResourceFromPipelineRuntime(runtime v1alpha1.Runtime) (RuntimeResource, error) {
	pipelineRuntime, ok := runtime.(*v1alpha1.ProjectPipelineRuntime)
	if !ok {
		return RuntimeResource{}, fmt.Errorf("runtime is not a pipeline runtime")
	}

	return RuntimeResource{
		Spaces:    utils.NewStringSet(pipelineRuntime.GetNamespaces()...),
		CodeRepos: utils.NewStringSet(pipelineRuntime.Spec.PipelineSource),
	}, nil
}

type AccountUsage struct {
	Accounts map[string]AccountResource `yaml:"accounts"`
}

func (au *AccountUsage) AddOrUpdateRuntime(accountName string, runtime v1alpha1.Runtime) error {
	runtimeResource, err := GetRuntimeResourceFunctionMap[runtime.GetRuntimeType()](runtime)
	if err != nil {
		return fmt.Errorf("get runtime resource failed: %w", err)
	}

	if au.Accounts == nil {
		au.Accounts = map[string]AccountResource{}
	}

	_, ok := au.Accounts[accountName]
	if !ok {
		au.Accounts[accountName] = AccountResource{
			Runtimes: map[string]RuntimeResource{},
		}
	}

	au.Accounts[accountName].Runtimes[runtime.GetName()] = runtimeResource
	return nil
}

func (au *AccountUsage) DeleteRuntime(accountName string, runtime v1alpha1.Runtime) {
	if au.Accounts == nil {
		return
	}
	_, ok := au.Accounts[accountName]
	if !ok {
		return
	}
	delete(au.Accounts[accountName].Runtimes, runtime.GetName())

	if len(au.Accounts[accountName].Runtimes) == 0 {
		delete(au.Accounts, accountName)
	}
}

type AccountResource struct {
	Runtimes map[string]RuntimeResource `yaml:"runtimes"`
}

type RuntimeResource struct {
	Spaces    utils.StringSet `yaml:"spaces"`
	CodeRepos utils.StringSet `yaml:"codeRepos"`
}

type listOptions struct {
	ExcludedRuntimeNames utils.StringSet
}

func newListOptions(opts ...ListOption) listOptions {
	getOpts := &listOptions{
		ExcludedRuntimeNames: utils.NewStringSet(),
	}

	for _, fn := range opts {
		fn(getOpts)
	}

	return *getOpts
}

type ListOption func(*listOptions)

func ExcludedRuntimeNames(runtimeNames []string) ListOption {
	return func(gopt *listOptions) {
		gopt.ExcludedRuntimeNames = utils.NewStringSet(runtimeNames...)
	}
}

func (ar AccountResource) ListAccountSpaces(opts ...ListOption) utils.StringSet {
	getOpts := newListOptions(opts...)

	spaceSet := utils.NewStringSet()
	for name, runtime := range ar.Runtimes {
		if getOpts.ExcludedRuntimeNames.Has(name) {
			continue
		}
		spaceSet.Set = spaceSet.Union(runtime.Spaces.Set)
	}
	return spaceSet
}

func (ar *AccountResource) ListAccountCodeRepos(opts ...ListOption) utils.StringSet {
	getOpts := newListOptions(opts...)

	codeRepoSet := utils.NewStringSet()
	for name, runtime := range ar.Runtimes {
		if getOpts.ExcludedRuntimeNames.Has(name) {
			continue
		}
		codeRepoSet.Set = codeRepoSet.Union(runtime.CodeRepos.Set)
	}
	return codeRepoSet
}
