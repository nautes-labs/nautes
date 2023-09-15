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
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nautescrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var mgr manager.Manager
var cancel context.CancelFunc
var ctx context.Context
var k8sClientFromOperator client.Client
var logger = logf.Log.WithName("base-operator-test")
var nautesConfigPath = "/tmp/nautes.config"
var nautesNamespace = "default"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	var use = false
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		UseExistingCluster:    &use,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = os.Setenv("NAUTESCONFIGPATH", nautesConfigPath)
	Expect(err).NotTo(HaveOccurred())

	configString := `
nautes:
  namespace: "default"
`
	err = os.WriteFile(nautesConfigPath, []byte(configString), 0600)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	err = nautescrd.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Port:   9443,
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	k8sClientFromOperator = mgr.GetClient()

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
	err = os.Remove(nautesConfigPath)
	Expect(err).Should(BeNil())
})

func randNum() string {
	return fmt.Sprintf("%04d", rand.Intn(9999))
}

func waitForIndexUpdate(obj client.ObjectList, selector fields.Selector) error {
	listOpts := &client.ListOptions{
		FieldSelector: selector,
	}

	for i := 0; i < 10; i++ {
		if err := k8sClientFromOperator.List(context.Background(), obj, listOpts); err != nil {
			continue
		}
		objList := reflect.ValueOf(obj).Elem()
		if objList.FieldByName("Items").Len() != 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("waiting for index updated time out")
}

func waitForCondition(obj client.Object, conditionType string) (*metav1.Condition, error) {
	var lastConditionTime time.Time
	var err error

	startTime := time.Now()
	logger.Info("Start to wait condition.", "StartTime", startTime.String())

	for i := 0; i < 10; i++ {
		reflect.ValueOf(obj).Elem().FieldByName("Status").FieldByName("Conditions").SetZero()
		err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return nil, err
		}

		ptrConditions := reflect.ValueOf(obj).Elem().FieldByName("Status").FieldByName("Conditions").Interface()
		conditions, ok := ptrConditions.([]metav1.Condition)
		if !ok {
			return nil, fmt.Errorf("get conditions from obj failed")
		}
		conditions = nautescrd.GetConditions(conditions,
			map[string]bool{
				conditionType: true,
			},
		)

		if len(conditions) != 0 {
			lastConditionTime = conditions[0].LastTransitionTime.Time
			// Avoid ending earlier than starting due to missing decimal points.
			if startTime.Before(lastConditionTime.Add(time.Second)) {
				logger.Info("Wait for condition updated over.", "LastUpdateTime", lastConditionTime.String())
				// Avoid update twice in one second.
				time.Sleep(time.Second * 1)
				return &conditions[0], nil
			}
		}
		time.Sleep(time.Second * 1)
	}

	logger.V(1).Info("wait for condition failed", "startTime", startTime, "lastConditionTime", lastConditionTime)
	return nil, fmt.Errorf("wait for condition timeout")
}

func waitForDelete(obj client.Object) error {
	clean := false

	for i := 0; i < 10; i++ {
		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
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
