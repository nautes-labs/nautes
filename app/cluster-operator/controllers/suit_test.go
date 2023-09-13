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

package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	nautesconfig "github.com/nautes-labs/nautes/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	factory "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/factory"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var mgr manager.Manager
var ctx context.Context
var cancel context.CancelFunc
var logger = logf.Log.WithName("cluster-controller-test")

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster operator")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())
	By("bootstrapping test environment")

	crdPath := filepath.Join("..", "..", "..", "config", "crd", "bases")
	_, err := os.Stat(crdPath)
	Expect(err).NotTo(HaveOccurred())

	var use = false
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &use,
	}

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = clustercrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	toCreateNamespace := "nautes"
	if isCreateNamespace(k8sClient, toCreateNamespace) {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: toCreateNamespace}}
		err = k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())
	}

	factory.GetFactory().AddNewMethod("mock", newMock)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nautes-configs",
			Namespace: "nautes",
		},
		Data: map[string]string{"config": `
secret:
  repoType: mock
`},
	}
	err = k8sClient.Create(ctx, cm)
	Expect(err).NotTo(HaveOccurred())

	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Port:   9443,
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Configs: nautesconfig.NautesConfigs{
			Name:      "nautes-configs",
			Namespace: "nautes",
		},
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func WaitForCondition(target *clustercrd.Cluster, startTime time.Time, conditionType string) (*metav1.Condition, error) {
	cluster := &clustercrd.Cluster{}
	var lastConditionTime time.Time
	var err error

	for i := 0; i < 30; i++ {
		err = k8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: target.Namespace,
			Name:      target.Name,
		}, cluster)
		if err != nil {
			return nil, err
		}

		conditions := cluster.Status.GetConditions(
			map[string]bool{
				conditionType: true,
			},
		)

		if len(conditions) != 0 {
			lastConditionTime = conditions[0].LastTransitionTime.Time
			// Avoid ending earlier than starting due to missing decimal points
			if startTime.Before(lastConditionTime.Add(time.Second)) {
				return &conditions[0], nil
			}
		}
		time.Sleep(time.Second * 1)
	}

	logger.V(1).Info("wait for condition failed", "startTime", startTime, "lastConditionTime", lastConditionTime)
	return nil, fmt.Errorf("wait for condition timeout")
}

func WaitForDelete(target *clustercrd.Cluster) error {
	clean := false

	cluster := &clustercrd.Cluster{}
	for i := 0; i < 30; i++ {
		err := k8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: target.Namespace,
			Name:      target.Name,
		}, cluster)
		if err != nil {
			clean = true
			break
		}
		time.Sleep(time.Second)
	}
	if !clean {
		return fmt.Errorf("wait for delete timeout")
	}
	return nil
}

var wantErr error
var wantResult *secretclient.SyncResult

type mockSyncer struct{}

func newMock(context.Context, *nautesconfig.Config, client.Client) (secretclient.SecretClient, error) {
	return &mockSyncer{}, nil
}
func (m mockSyncer) Sync(ctx context.Context, cluster, lastCluster *clustercrd.Cluster) (*secretclient.SyncResult, error) {
	return wantResult, wantErr
}

func (m mockSyncer) Delete(ctx context.Context, cluster *clustercrd.Cluster) error {
	return wantErr
}

func (m mockSyncer) GetKubeConfig(ctx context.Context, cluster *clustercrd.Cluster) (string, error) {
	apiCfg := api.Config{
		Clusters: map[string]*api.Cluster{
			"default": {
				Server:                   cfg.Host,
				InsecureSkipTLSVerify:    cfg.Insecure,
				CertificateAuthorityData: cfg.CAData,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"default": {
				ClientCertificateData: cfg.CertData,
				ClientKeyData:         cfg.KeyData,
			},
		},
		Contexts: map[string]*api.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		CurrentContext: "default",
	}
	kubeconfig, err := clientcmd.Write(apiCfg)
	Expect(err).Should(BeNil())

	if wantErr != nil {
		return "", wantErr
	}
	return string(kubeconfig), nil
}

func (m mockSyncer) Logout() {}

func isCreateNamespace(k8sClient client.Client, createdNamespace string) bool {
	namespaces := &corev1.NamespaceList{}
	err := k8sClient.List(context.Background(), namespaces)
	Expect(err).NotTo(HaveOccurred())

	for _, namespace := range namespaces.Items {
		if namespace.Name == createdNamespace {
			return false
		}
	}

	return true
}
