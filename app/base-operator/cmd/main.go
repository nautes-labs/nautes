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

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nautescfg "github.com/nautes-labs/nautes/pkg/nautesconfigs"

	productsyncer "github.com/nautes-labs/nautes/app/base-operator/internal/syncer/product"
	productprovidersyncer "github.com/nautes-labs/nautes/app/base-operator/internal/syncer/productprovider"

	nautesv1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"

	"github.com/nautes-labs/nautes/app/base-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nautesv1alpha1.AddToScheme(scheme))
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
		Namespace:              globalConfigNamespace,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0afe6787.resource.nautes.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cfg := nautescfg.NautesConfigs{
		Namespace: globalConfigNamespace,
		Name:      globalConfigName,
	}

	providerSyncer := &productprovidersyncer.ProductProviderSyncer{
		NautesConfig: cfg,
		Rest:         mgr.GetConfig(),
	}
	if err := providerSyncer.Setup(); err != nil {
		setupLog.Error(err, "init product provider syncer failed")
		os.Exit(1)
	}

	syncer := &productsyncer.ProductSyncer{
		NautesConfig: cfg,
		Rest:         mgr.GetConfig(),
	}

	if err := syncer.Setup(); err != nil {
		setupLog.Error(err, "init product syncer failed")
		os.Exit(1)
	}

	if err = (&controllers.ProductProviderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Syncer: providerSyncer,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ProductoProvider")
		os.Exit(1)
	}

	if err = (&controllers.ProductReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Syncer: syncer,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Product")
		os.Exit(1)
	}

	if err = (&controllers.ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
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
