package tekton_test

import (
	"context"
	"testing"

	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/component"
	shared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/tekton"

	pluginshared "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/shared"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTekton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tekton Suite")
}

var fakeHook *pluginshared.Hook

type mockSecretSync struct{}

func (mss *mockSecretSync) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) GetComponentMachineAccount() *component.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) CreateSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

func (mss *mockSecretSync) RemoveSecret(ctx context.Context, secretReq component.SecretRequest) error {
	return nil
}

func ConvertInterfaceToRequest(reqs []interface{}) []shared.ResourceRequest {
	resReqs := make([]shared.ResourceRequest, len(reqs))
	for i := range reqs {
		req := reqs[i].(shared.ResourceRequest)
		resReqs[i] = req
	}
	return resReqs
}

type mockPluginManager struct{}

// GetHookFactory will return a 'HookFactory' that can implement the hook based on the pipeline type and hook type.
// If the corresponding plugin cannot be found, an error is returned.
func (mpm *mockPluginManager) GetHookFactory(pipelineType string, hookName string) (pluginshared.HookFactory, error) {
	return &mockPlugin{}, nil
}

type mockPlugin struct{}

func (mp *mockPlugin) GetPipelineType() (string, error) {
	return "", nil
}

func (mp *mockPlugin) GetHooksMetadata() ([]pluginshared.HookMetadata, error) {
	return nil, nil
}

func (mp *mockPlugin) BuildHook(hookName string, info pluginshared.HookBuildData) (*pluginshared.Hook, error) {
	return fakeHook, nil
}
