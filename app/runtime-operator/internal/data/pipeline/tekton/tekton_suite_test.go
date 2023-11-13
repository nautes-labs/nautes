package tekton_test

import (
	"context"
	"testing"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/pipeline/tekton"
	syncer "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/interface"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTekton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tekton Suite")
}

type mockSecretSync struct{}

func (mss *mockSecretSync) CleanUp() error {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) GetComponentMachineAccount() *syncer.MachineAccount {
	panic("not implemented") // TODO: Implement
}

func (mss *mockSecretSync) CreateSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	return nil
}

func (mss *mockSecretSync) RemoveSecret(ctx context.Context, secretReq syncer.SecretRequest) error {
	return nil
}

func ConvertInterfaceToRequest(reqs []interface{}) []tekton.ResourceRequest {
	resReqs := make([]tekton.ResourceRequest, len(reqs))
	for i := range reqs {
		req := reqs[i].(tekton.ResourceRequest)
		resReqs[i] = req
	}
	return resReqs
}
