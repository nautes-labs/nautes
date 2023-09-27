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

package v1alpha1

import "time"

const (
	DefaultSyncRetryMaxDuration time.Duration = 180000000000 // 3m0s
	DefaultSyncRetryDuration    time.Duration = 5000000000   // 5s
	DefaultSyncRetryFactor                    = int64(2)
	// ResourcesFinalizerName is the finalizer value which we inject to finalize deletion of an application
	ResourcesFinalizerName string = "resources-finalizer.argocd.argoproj.io"

	// ForegroundPropagationPolicyFinalizer is the finalizer we inject to delete application with foreground propagation policy
	ForegroundPropagationPolicyFinalizer string = "resources-finalizer.argocd.argoproj.io/foreground"

	// BackgroundPropagationPolicyFinalizer is the finalizer we inject to delete application with background propagation policy
	BackgroundPropagationPolicyFinalizer = "resources-finalizer.argocd.argoproj.io/background"

	// DefaultAppProjectName contains name of 'default' app project, which is available in every Argo CD installation
	DefaultAppProjectName = "default"

	// RevisionHistoryLimit is the max number of successful sync to keep in history
	RevisionHistoryLimit = 10

	// KubernetesInternalAPIServerAddr is address of the k8s API server when accessing internal to the cluster
	KubernetesInternalAPIServerAddr = "https://kubernetes.default.svc"
)
