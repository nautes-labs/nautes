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

// Package controllers use to set k8s cluster info to secret store
package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	clustercrd "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	operatorerror "github.com/nautes-labs/nautes/app/cluster-operator/pkg/error"
	factory "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/factory"
	secretclient "github.com/nautes-labs/nautes/app/cluster-operator/pkg/secretclient/interface"
	nautesconfig "github.com/nautes-labs/nautes/pkg/nautesconfigs"
)

const (
	clusterFinalizerName                     = "cluster.nautes.resource.nautes.io/finalizers"
	clusterConditionTypeSecretStore          = "SecretStoreReady"
	clusterConditionTypeEntryPointCollection = "EntryPointCollected"
	clusterConditionReason                   = "RegularUpdate"
)

var (
	LabelClusterEntryPoint = map[string]string{"resource.nautes.io/Usage": "Entrypoint"}
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Configs nautesconfig.NautesConfigs
}

//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=clusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=coderepoes,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=environments,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=deploymentruntimes,verbs=get;list;watch
//+kubebuilder:rbac:groups=nautes.resource.nautes.io,resources=projectpipelineruntimes,verbs=get;list;watch
//+kubebuilder:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cluster := &clustercrd.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !cluster.DeletionTimestamp.IsZero() {
		if err := r.deleteCluster(ctx, *cluster); err != nil {
			setStatus(cluster, nil, err)
			if err := r.Status().Update(ctx, cluster); err != nil {
				logger.Error(err, "update condition failed")
			}
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(cluster, clusterFinalizerName)
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(cluster, clusterFinalizerName) {
		controllerutil.AddFinalizer(cluster, clusterFinalizerName)
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	result, err := r.syncCluster(ctx, *cluster)
	entryPoints, serviceErrs, collectEntryPointErr := r.getClusterEntryPort(ctx, cluster)

	if err := r.updateStatus(ctx, req.NamespacedName,
		func(c *clustercrd.Cluster) {
			setStatus(c, result, err)
			setEntryPointCollectionStatus(c, entryPoints, serviceErrs, collectEntryPointErr)
		}); err != nil {
		return ctrl.Result{}, err
	}

	requeueAfter := time.Second * 60
	if err != nil {
		var operatorErr operatorerror.Errors
		if errors.As(err, &operatorErr) {
			requeueAfter = operatorErr.RequeueAfter
			logger.Error(err, "get cluster info failed")
		} else {
			return ctrl.Result{}, err
		}
	}

	logger.V(1).Info("reconcile finish", "RequeueAfter", fmt.Sprintf("%gs", requeueAfter.Seconds()))
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustercrd.Cluster{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *ClusterReconciler) newSecretClient(ctx context.Context) (secretclient.SecretClient, error) {
	cfg, err := r.Configs.GetConfigByClient(r.Client)
	if err != nil {
		return nil, fmt.Errorf("read error global config failed: %w", err)
	}

	secClient, err := factory.GetFactory().NewSecretClient(ctx, cfg, r.Client)
	if err != nil {
		return nil, fmt.Errorf("get secret client failed: %w", err)
	}

	return secClient, nil
}

func (r *ClusterReconciler) getClusterEntryPort(ctx context.Context, cluster *clustercrd.Cluster) (map[string]clustercrd.ClusterEntryPoint, []error, error) {
	if cluster == nil {
		return nil, nil, fmt.Errorf("cluster is nil")
	}
	if cluster.Spec.Usage == clustercrd.CLUSTER_USAGE_HOST {
		return nil, nil, nil
	}

	k8sClient, err := r.getEntryPointClusterClient(ctx, cluster)
	if err != nil {
		return nil, nil, fmt.Errorf("get cluster client failed: %w", err)
	}

	svcs := &corev1.ServiceList{}
	listOpt := client.MatchingLabels(LabelClusterEntryPoint)
	if err := k8sClient.List(ctx, svcs, listOpt); err != nil {
		return nil, nil, fmt.Errorf("list svc failed: %w", err)
	}

	entryPoints := make(map[string]clustercrd.ClusterEntryPoint)
	errs := []error{}
	for _, svc := range svcs.Items {
		entrypoint, err := findTargetPort(svc)
		if err != nil {
			log.FromContext(ctx).Error(err, "can not get port from service", "ServiceName", svc.Name, "ServiceNamespace", svc.Namespace)
			errs = append(errs, fmt.Errorf("can no get port from service %s in namespace %s: %w;", svc.Name, svc.Namespace, err))
			continue
		}
		entryPoints[svc.Name] = *entrypoint
	}

	return entryPoints, errs, nil
}

// If cluster is virtual, it will return host cluster client, else it will return cluster client itself.
func (r *ClusterReconciler) getEntryPointClusterClient(ctx context.Context, cluster *clustercrd.Cluster) (client.Client, error) {
	secClient, err := r.newSecretClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get secret client failed: %w", err)
	}
	defer secClient.Logout()

	var targetCluster *clustercrd.Cluster
	if cluster.Spec.ClusterType == clustercrd.CLUSTER_TYPE_VIRTUAL {
		hostCluster := &clustercrd.Cluster{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Spec.HostCluster,
		}, hostCluster)
		if err != nil {
			return nil, err
		}
		targetCluster = hostCluster
	} else {
		targetCluster = cluster
	}

	kubeconfig, err := secClient.GetKubeConfig(ctx, targetCluster)
	if err != nil {
		return nil, fmt.Errorf("get cluster accessinfo failed: %w", err)
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("get cluster accessinfo failed: %w", err)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("get rest config failed: %w", err)
	}

	return client.New(restConfig, client.Options{})
}

const (
	annotationHTTPEntrypoint  = "cluster-operator.nautes.io/http-entrypoint"
	annotationHTTPSEntrypoint = "cluster-operator.nautes.io/https-entrypoint"
)

func findTargetPort(svc corev1.Service) (*clustercrd.ClusterEntryPoint, error) {
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		return findTargetPortLoadBalancer(svc)
	case corev1.ServiceTypeNodePort:
		return findTargetPortNodePort(svc)
	default:
		return nil, fmt.Errorf("service type can not be entry point")
	}
}

func findTargetPortLoadBalancer(svc corev1.Service) (*clustercrd.ClusterEntryPoint, error) {
	entryPoint := &clustercrd.ClusterEntryPoint{
		Type: clustercrd.ServiceTypeLoadBalancer,
	}
	httpName, hasHTTPKey := svc.Annotations[annotationHTTPEntrypoint]
	httpsName, hasHTTPSKey := svc.Annotations[annotationHTTPSEntrypoint]

	for _, port := range svc.Spec.Ports {
		if hasHTTPKey && port.Name == httpName {
			entryPoint.HTTPPort = port.Port
		}
		if hasHTTPSKey && port.Name == httpsName {
			entryPoint.HTTPSPort = port.Port
		}
	}

	if (!hasHTTPKey && !hasHTTPSKey) ||
		(hasHTTPKey && entryPoint.HTTPPort == 0) ||
		(hasHTTPSKey && entryPoint.HTTPSPort == 0) {
		return nil, fmt.Errorf("can not find any port in service")
	}

	return entryPoint, nil
}

func findTargetPortNodePort(svc corev1.Service) (*clustercrd.ClusterEntryPoint, error) {
	entryPoint := &clustercrd.ClusterEntryPoint{
		Type: clustercrd.ServiceTypeNodePort,
	}
	httpName, hasHTTPKey := svc.Annotations[annotationHTTPEntrypoint]
	httpsName, hasHTTPSKey := svc.Annotations[annotationHTTPSEntrypoint]

	for _, port := range svc.Spec.Ports {
		if hasHTTPKey && port.Name == httpName {
			entryPoint.HTTPPort = port.NodePort
		}
		if hasHTTPSKey && port.Name == httpsName {
			entryPoint.HTTPSPort = port.NodePort
		}
	}

	if (!hasHTTPKey && !hasHTTPSKey) ||
		(hasHTTPKey && entryPoint.HTTPPort == 0) ||
		(hasHTTPSKey && entryPoint.HTTPSPort == 0) {
		return nil, fmt.Errorf("can not find any port in service")
	}

	return entryPoint, nil
}

func (r *ClusterReconciler) syncCluster(ctx context.Context, cluster clustercrd.Cluster) (*clustercrd.MgtClusterAuthStatus, error) {
	lastCluster, err := getSyncHistory(&cluster)
	if err != nil {
		return nil, err
	}

	if err := cluster.ValidateCluster(ctx, lastCluster, r.Client, false); err != nil {
		return nil, err
	}

	secretClient, err := r.newSecretClient(ctx)
	if err != nil {
		return nil, err
	}

	result, err := secretClient.Sync(ctx, &cluster, lastCluster)
	if err != nil {
		return nil, fmt.Errorf("sync cluster info to secret store failed: %w", err)
	}

	return &clustercrd.MgtClusterAuthStatus{
		LastSuccessSpec: cluster.SpecToJsonString(),
		SecretID:        result.SecretID,
	}, nil
}

func (r *ClusterReconciler) deleteCluster(ctx context.Context, cluster clustercrd.Cluster) error {
	if !controllerutil.ContainsFinalizer(&cluster, clusterFinalizerName) {
		return nil
	}

	if err := cluster.ValidateCluster(ctx, nil, r.Client, true); err != nil {
		return err
	}

	lastCluster, err := getSyncHistory(&cluster)
	if err != nil {
		return err
	}

	secretClient, err := r.newSecretClient(ctx)
	if err != nil {
		return err
	}

	err = secretClient.Delete(ctx, lastCluster)
	if err != nil {
		return fmt.Errorf("delete cluster info from secret store failed: %w", err)
	}
	return nil
}

type setStatusFunction func(*clustercrd.Cluster)

func (r *ClusterReconciler) updateStatus(ctx context.Context, key types.NamespacedName, setStatus setStatusFunction) error {
	cluster := &clustercrd.Cluster{}
	if err := r.Get(ctx, key, cluster); err != nil {
		return err
	}

	setStatus(cluster)

	return r.Status().Update(ctx, cluster)
}

func setStatus(cluster *clustercrd.Cluster, result *clustercrd.MgtClusterAuthStatus, err error) {
	if err != nil {
		condition := metav1.Condition{
			Type:    clusterConditionTypeSecretStore,
			Status:  "False",
			Reason:  clusterConditionReason,
			Message: err.Error(),
		}
		cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{clusterConditionTypeSecretStore: true})
	} else {
		condition := metav1.Condition{
			Type:   clusterConditionTypeSecretStore,
			Status: "True",
			Reason: clusterConditionReason,
		}
		cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{clusterConditionTypeSecretStore: true})

		if result != nil {
			cluster.Status.MgtAuthStatus = result
			conditions := cluster.Status.GetConditions(map[string]bool{clusterConditionTypeSecretStore: true})
			if len(conditions) == 1 {
				cluster.Status.MgtAuthStatus.LastSuccessTime = conditions[0].LastTransitionTime
			}
		}
	}
}

func setEntryPointCollectionStatus(cluster *clustercrd.Cluster, entryPoints map[string]clustercrd.ClusterEntryPoint, errs []error, err error) {
	if err != nil {
		condition := metav1.Condition{
			Type:    clusterConditionTypeEntryPointCollection,
			Status:  "False",
			Reason:  clusterConditionReason,
			Message: err.Error(),
		}
		cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{clusterConditionTypeEntryPointCollection: true})
	} else if len(errs) != 0 {
		condition := metav1.Condition{
			Type:    clusterConditionTypeEntryPointCollection,
			Status:  "False",
			Reason:  clusterConditionReason,
			Message: fmt.Sprintf("some get entrypoint failed: %v", errs),
		}
		cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{clusterConditionTypeEntryPointCollection: true})
		cluster.Status.EntryPoints = entryPoints
	} else {
		condition := metav1.Condition{
			Type:   clusterConditionTypeEntryPointCollection,
			Status: "True",
			Reason: clusterConditionReason,
		}
		cluster.Status.SetConditions([]metav1.Condition{condition}, map[string]bool{clusterConditionTypeEntryPointCollection: true})
		cluster.Status.EntryPoints = entryPoints
	}
}

func getSyncHistory(cluster *clustercrd.Cluster) (*clustercrd.Cluster, error) {
	if cluster.Status.MgtAuthStatus == nil {
		return nil, nil
	}

	lastCluster, err := clustercrd.GetClusterFromString(cluster.Name, cluster.Namespace, cluster.Status.MgtAuthStatus.LastSuccessSpec)
	if err != nil {
		return nil, fmt.Errorf("get history from status failed: %w", err)
	}

	return lastCluster, nil
}
