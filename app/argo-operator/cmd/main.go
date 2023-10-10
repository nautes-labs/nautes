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
	"flag"
	"os"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/argo-operator/controllers/cluster"
	"github.com/nautes-labs/nautes/app/argo-operator/controllers/coderepo"
	argocd "github.com/nautes-labs/nautes/app/argo-operator/pkg/argocd"
	"github.com/nautes-labs/nautes/app/argo-operator/pkg/secret"
	zaplog "github.com/nautes-labs/nautes/pkg/log/zap"
	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(resourcev1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var argocdServer string
	var globalConfigNamespace string
	var globalConfigName string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&argocdServer, "argocd-server", "https://argocd-server.argocd.svc", "The address is argocd server.")
	flag.StringVar(&globalConfigName, "global-config-name", "nautes-configs", "The resources name of global config.")
	flag.StringVar(&globalConfigNamespace, "global-config-namespace", "nautes", "The namespace of global config in.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	logger := zaplog.New()
	ctrl.SetLogger(logger)

	setupLog.Info("global config", "namespace", globalConfigNamespace, "file name", globalConfigName)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "457c014f.nautes.io",
		Namespace:              globalConfigNamespace,
		ClientDisableCacheFor: []client.Object{
			&resourcev1alpha1.Cluster{},
		},
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cluserSecret, err := secret.NewVaultClient()
	if err != nil {
		setupLog.Error(err, "failed to init vault client")
		os.Exit(1)
	}

	if err = (&coderepo.CodeRepoReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Argocd:                argocd.NewArgocd(argocdServer),
		Secret:                cluserSecret,
		Log:                   ctrl.Log.WithName("codeRepo controller log"),
		GlobalConfigNamespace: globalConfigNamespace,
		GlobalConfigName:      globalConfigName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CodeRepo")
		os.Exit(1)
	}

	codeRepoSecret, err := secret.NewVaultClient()
	if err != nil {
		setupLog.Error(err, "failed to init vault client")
		os.Exit(1)
	}

	if err = (&cluster.ClusterReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Argocd:                argocd.NewArgocd(argocdServer),
		Secret:                codeRepoSecret,
		Log:                   ctrl.Log.WithName("cluster controller log"),
		GlobalConfigNamespace: globalConfigNamespace,
		GlobalConfigName:      globalConfigName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
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
