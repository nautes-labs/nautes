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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/deployment/argocd"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/eventlistener/argoevent"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/multitenant/hnc"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/pipeline/tekton"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/secretmanagement/vault"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/data/secretsync/externalsecret"
	"github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/handler"
	syncerv2 "github.com/nautes-labs/nautes/app/runtime-operator/internal/syncer/v2/task"
	pluginmanager "github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/manager"

	"github.com/nautes-labs/nautes/app/runtime-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	syncerv2.NewFunctionMapDeployment["argocd"] = argocd.NewArgoCD
	syncerv2.NewFunctionMapMultiTenant["hnc"] = hnc.NewHNC
	syncerv2.NewFunctionMapSecretManagement["vault"] = vault.NewSecretManagement
	syncerv2.NewFunctionMapEventListener["argo-events"] = argoevent.NewArgoEvent
	syncerv2.NewFunctionMapSecretSync["external-secrets"] = externalsecret.NewExternalSecret
	syncerv2.ComponentFactoryMapPipeline["tekton"] = tekton.TektonFactory
	syncerv2.NewFunctionHandler = handler.NewHandler

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var globalConfigName string
	var globalConfigNamespace string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&globalConfigName, "global-config-name", "nautes-configs", "The resources name of global config.")
	flag.StringVar(&globalConfigNamespace, "global-config-namespace", "nautes", "The namespace of global config in.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "af2185dd.resource.nautes.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	plgMgr, err := pluginmanager.NewPluginManagement(&pluginmanager.NewOptions{
		Client: mgr.GetClient(),
	})
	if err != nil {
		setupLog.Error(err, "get plugin management failed")
		os.Exit(1)
	}
	go func() {
		err := plgMgr.Run()
		if err != nil {
			setupLog.Error(err, "run plugin management failed")
			os.Exit(1)
		}
	}()
	defer plgMgr.Kill()

	ctx := context.TODO()
	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.CodeRepo{}, v1alpha1.SelectFieldMetaDataName, func(obj client.Object) []string {
		return []string{obj.GetName()}
	}); err != nil {
		setupLog.Error(err, "add index field failed")
		os.Exit(1)
	}
	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.Cluster{}, v1alpha1.SelectFieldMetaDataName, func(obj client.Object) []string {
		return []string{obj.GetName()}
	}); err != nil {
		setupLog.Error(err, "add index field failed")
		os.Exit(1)
	}
	if err := mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.CodeRepoBinding{}, v1alpha1.SelectFieldCodeRepoBindingProductAndRepo, func(obj client.Object) []string {
		binding := obj.(*v1alpha1.CodeRepoBinding)
		if binding.Spec.Product == "" || binding.Spec.CodeRepo == "" {
			return nil
		}
		return []string{fmt.Sprintf("%s/%s", binding.Spec.Product, binding.Spec.CodeRepo)}
	}); err != nil {
		setupLog.Error(err, "add index field failed")
		os.Exit(1)
	}

	if err = (&controllers.DeploymentRuntimeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Syncer: syncerv2.Syncer{
			KubernetesClient: mgr.GetClient(),
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DeploymentRuntime")
		os.Exit(1)
	}

	if err = (&controllers.ProjectPipelineRuntimeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Syncer: syncerv2.Syncer{
			KubernetesClient: mgr.GetClient(),
			PluginMgr:        plgMgr,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProjectPipelineRuntime")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
