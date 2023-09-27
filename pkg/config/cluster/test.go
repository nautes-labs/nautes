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

package cluster

import (
	"os"
)

func SetClusterValidateConfig() error {
	path, err := createTempFile(ThirdPartComponentsFile, mockThirdPartComponentsData())
	if err != nil {
		return err
	}

	err = os.Setenv(EnvthirdPartComponents, path)
	if err != nil {
		return err
	}

	path, err = createTempFile(ComponentCategoryFile, mockcomponentsOfClusterTypeData())
	if err != nil {
		return err
	}

	err = os.Setenv(EnvComponentCategoryDefinition, path)
	if err != nil {
		return err
	}

	path, err = createTempFile(ClusterCommonConfigFile, mockClusterCommonConfig())
	if err != nil {
		return err
	}

	err = os.Setenv(EnvClusterCommonConfig, path)
	if err != nil {
		return err
	}

	return nil
}

func DeleteValidateConfig() error {
	err := deleteFileByEnv(EnvthirdPartComponents)
	if err != nil {
		return err
	}

	err = deleteFileByEnv(EnvComponentCategoryDefinition)
	if err != nil {
		return err
	}

	err = deleteFileByEnv(EnvClusterCommonConfig)
	if err != nil {
		return err
	}

	return nil
}

func createTempFile(name, content string) (string, error) {
	tempDir := os.TempDir()

	tempFile, err := os.CreateTemp(tempDir, name)
	if err != nil {
		return "", err
	}

	defer tempFile.Close()

	_, err = tempFile.WriteString(content)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func deleteFileByEnv(env string) error {
	filePath := os.Getenv(env)

	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

func mockThirdPartComponentsData() string {
	return `
- name: tekton
  namespace: tekton-pipelines
  type: pipeline
  default: true
  general: false
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/tekton
      - cluster_templates/runtimes/_RUNTIME_/production/tekton-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/production/tekton-cluster-tasks-app.yaml
- name: argocd
  namespace: argocd
  type: deployment
  default: true
  general: true
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/argocd
      - cluster_templates/tenant/production/runtime-argocd-appset.yaml
- name: argo-events
  namespace: argo-events
  type: eventListener
  default: true
  general: false
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/production/argo-events-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/argoevents
- name: argo-rollouts
  namespace: argo-rollouts
  type: progressiveDelivery
  default: true
  general: false
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/production/argo-rollouts-app.yaml
- name: hnc
  namespace: hnc-system
  type: multiTenant
  default: true
  general: true
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/production/hnc-app.yaml
- name: cert-manager
  namespace: cert-manager
  type: certManagement
  default: true
  general: true
  properties:
  installPath:
    '*':
      - ""
- name: vault
  namespace: vault
  type: secretManagement
  default: true
  general: true
  properties:
  installPath:
    'worker&physical':
      - cluster_templates/runtimes/_RUNTIME_/production/vault-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/initial/overlays/production/rbac-vault.yaml
    'worker&virtual':
      - cluster_templates/runtimes/_RUNTIME_/production/vault-app.yaml
- name: external-secrets
  namespace: external-secrets
  type: secretSync
  default: true
  general: true
  properties:
  installPath:
    '*':
      - cluster_templates/runtimes/_RUNTIME_/production/external-secrets-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/production/external-secrets-resources-app.yaml
- name: oauth2-proxy
  namespace: oauth2-proxy
  type: oauthProxy
  default: true
  general: false
  properties:
  installPath:
    'host&physical':
    - cluster_templates/host-clusters/_HOST_CLUSTER/production/oauth2-proxy-app.yaml
    - cluster_templates/host-clusters/_HOST_CLUSTER/production/oauth2-proxy-resources-app.yaml
    - cluster_templates/host-clusters/_HOST_CLUSTER/oauth2-proxy
    'worker&physical':
    - cluster_templates/runtimes/_RUNTIME_/production/oauth2-proxy-app.yaml
    - cluster_templates/runtimes/_RUNTIME_/production/oauth2-proxy-resources-app.yaml
    - cluster_templates/runtimes/_RUNTIME_/oauth2-proxy
- name: traefik
  namespace: traefik
  type: gateway
  default: true
  general: true
  properties:
    - name: httpNodePort
      type: number
      regexPattern: ^[0-9]+$
      required: true
      default: 30080
    - name: httpsNodePort
      type: number
      regexPattern: ^[0-9]+$
      required: true
      default: 30443
  installPath:
    'host&physical':
      - cluster_templates/host-clusters/_HOST_CLUSTER/production/traefik-app.yaml
      - cluster_templates/host-clusters/_HOST_CLUSTER/production/traefik-resources-app.yaml
      - cluster_templates/host-clusters/_HOST_CLUSTER/traefik
    'worker&physical':
      - cluster_templates/runtimes/_RUNTIME_/production/traefik-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/production/traefik-resources-app.yaml
      - cluster_templates/runtimes/_RUNTIME_/traefik
    'worker&virtual':
      - cluster_templates/host-clusters/_HOST_CLUSTER/traefik/overlays/production/middleware-oauth.yaml`
}

func mockcomponentsOfClusterTypeData() string {
	return `
host:
  - oauthProxy
  - gateway
worker:
  virtual:
      deployment:
        - multiTenant
        - secretSync
        - deployment
        - progressiveDelivery
      pipeline:
        - multiTenant
        - secretSync
        - gateway
        - deployment
        - eventListener
        - pipeline
  physical:
      deployment:
        - multiTenant
        - secretSync
        - gateway
        - deployment
        - progressiveDelivery
      pipeline:
        - multiTenant
        - secretSync
        - gateway
        - deployment
        - eventListener
        - pipeline
        - oauthProxy`
}

func mockClusterCommonConfig() string {
	return `
save:
  host:
      - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
      - cluster_templates/tenant/production/host-cluster-appset.yaml
  worker:
      physical:
          - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
          - cluster_templates/runtimes/_RUNTIME_/initial/overlays/production/kustomization.yaml
          - cluster_templates/runtimes/_RUNTIME_/initial/overlays/production/cm-nautes.yaml
          - cluster_templates/runtimes/_RUNTIME_/runtime-app.yaml
          - cluster_templates/runtimes/_RUNTIME_/runtime-project.yaml
          - cluster_templates/tenant/production/runtime-appset.yaml
          - cluster_templates/tenant/production/runtime-initial-appset.yaml
      virtual:
          - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
          - cluster_templates/runtimes/_RUNTIME_/initial/overlays/production/kustomization.yaml
          - cluster_templates/runtimes/_RUNTIME_/initial/overlays/production/cm-nautes.yaml
          - cluster_templates/host-clusters/_HOST_CLUSTER/vclusters
          - cluster_templates/host-clusters/_HOST_CLUSTER/production/vcluster-appset.yaml
          - cluster_templates/runtimes/_RUNTIME_/runtime-app.yaml
          - cluster_templates/runtimes/_RUNTIME_/runtime-project.yaml
          - cluster_templates/tenant/production/runtime-appset.yaml
          - cluster_templates/tenant/production/runtime-initial-appset.yaml

remove:
  host:
      - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
      - cluster_templates/tenant/production/host-cluster-appset.yaml
  worker:
      physical:
          - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
      virtual:
          - cluster_templates/nautes/overlays/production/clusters/kustomization.yaml
          - cluster_templates/host-clusters/_HOST_CLUSTER/production/vcluster-appset.yaml
  `
}
