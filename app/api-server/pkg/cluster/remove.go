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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"sigs.k8s.io/kustomize/api/types"
	yaml "sigs.k8s.io/yaml"
)

// Remove this function remove cluster configuration based on cluster type
func (cr *ClusterRegistration) Remove() error {
	nodes, err := cr.DeleteClusterByType()
	if err != nil {
		return err
	}

	err = cr.SaveClusterConfig(nodes)
	if err != nil {
		return err
	}

	err = cr.DeleteClusterToNautes()
	if err != nil {
		return err
	}

	err = cr.CleanupAppSetIfEmpty()
	if err != nil {
		return err
	}

	return nil
}

func (cr *ClusterRegistration) DeleteClusterByType() (*nodestree.Node, error) {
	var nodes nodestree.Node

	config, err := NewClusterFileIgnoreConfig(cr.ClusterTemplateRepoLocalPath)
	if err != nil {
		return nil, err
	}

	ignorePaths, ignoreFiles, err := cr.getIgnoreConfig(config)
	if err != nil {
		return nil, err
	}

	nodes, err = cr.LoadTemplateNodesTree(ignorePaths, ignoreFiles)
	if err != nil {
		return nil, err
	}

	err = cr.deleteRuntimeByType(&nodes)
	if err != nil {
		return nil, err
	}

	return &nodes, nil
}

func (cr *ClusterRegistration) getIgnoreConfig(config *ClusterFileIgnoreConfig) ([]string, []string, error) {
	switch cr.Usage {
	case _HostCluster:
		ignorePaths, ignoreFiles := config.GetRemoveHostClusterConfig()
		return ignorePaths, ignoreFiles, nil
	case _PhysicalDeploymentRuntime:
		ignorePaths, ignoreFiles := config.GetRemovePhysicalDeploymentRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _PhysicalProjectPipelineRuntime:
		ignorePaths, ignoreFiles := config.GetRemovePhysicalProjectPipelineRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _VirtualDeploymentRuntime:
		ignorePaths, ignoreFiles := config.GetRemoveVirtualDeploymentRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _VirtualProjectPipelineRuntime:
		ignorePaths, ignoreFiles := config.GetRemoveVirtualProjectPipelineRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	default:
		return nil, nil, errors.New("unknown cluster usage")
	}
}

func (cr *ClusterRegistration) deleteRuntimeByType(nodes *nodestree.Node) error {
	switch cr.Usage {
	case _HostCluster:
		return cr.DeleteHostCluster(nodes)
	case _VirtualDeploymentRuntime:
		return cr.DeleteRuntime(nodes)
	case _PhysicalDeploymentRuntime:
		return cr.DeleteRuntime(nodes)
	case _VirtualProjectPipelineRuntime:
		return cr.DeleteRuntime(nodes)
	case _PhysicalProjectPipelineRuntime:
		return cr.DeleteRuntime(nodes)
	default:
		return errors.New("unknown cluster usage")
	}
}

func (cr *ClusterRegistration) DeleteHostCluster(nodes *nodestree.Node) error {
	hostClusterDir := fmt.Sprintf("%s/%s", concatHostClustesrDir(cr.TenantConfigRepoLocalPath), cr.Cluster.Name)
	vclustersDir := fmt.Sprintf("%s/vclusters", hostClusterDir)
	exist := isExistDir(vclustersDir)
	if exist {
		return fmt.Errorf("unable to delete cluster %s because the host cluster is referenced by other virtual cluster", cr.Cluster.Name)
	}

	err := DeleteSpecifyDir(hostClusterDir)
	if err != nil {
		return err
	}

	err = cr.DeleteClusterToKustomization()
	if err != nil {
		return err
	}

	err = cr.GetAndDeleteHostClusterNames()
	if err != nil {
		return err
	}

	err = cr.Execute(nodes)
	if err != nil {
		return err
	}

	cr.ReplaceTemplatePathWithTenantRepositoryPath(nodes)

	return nil
}

func (cr *ClusterRegistration) DeleteClusterToKustomization() (err error) {
	kustomizationFilePath := fmt.Sprintf("%s/nautes/overlays/production/clusters/kustomization.yaml", cr.TenantConfigRepoLocalPath)
	if _, err := os.Stat(kustomizationFilePath); os.IsNotExist(err) {
		return err
	}

	bytes, err := os.ReadFile(kustomizationFilePath)
	if err != nil {
		return
	}
	var kustomization types.Kustomization
	err = yaml.Unmarshal(bytes, &kustomization)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.yaml", cr.Cluster.Name)
	if len(kustomization.Resources) > 0 {
		cr.ClusterResouceFiles = RemoveStringFromArray(kustomization.Resources, filename)
	}

	return nil
}

func (cr *ClusterRegistration) GetAndDeleteHostClusterNames() error {
	hostClusterAppsetFilePath := concatHostClusterAppsetFilePath(cr.TenantConfigRepoLocalPath)
	clusterNames, err := GetHostClusterNames(hostClusterAppsetFilePath)
	if err != nil {
		return err
	}

	cr.HostClusterNames = RemoveStringFromArray(clusterNames, cr.HostCluster.Name)

	return nil
}

// FilterDeletedClusters filter deleted clusters from vclusters and return new ones
func (cr *ClusterRegistration) FilterDeletedClusters() error {
	vclusterAppsetFilePath := fmt.Sprintf("%s/%s/%s", concatHostClustesrDir(cr.TenantConfigRepoLocalPath), cr.Vcluster.HostCluster.Name, _VclusterAppSetFile)
	vclusterNames, err := GetVclusterNames(vclusterAppsetFilePath)
	if err != nil {
		return err
	}

	cr.VclusterNames = RemoveStringFromArray(vclusterNames, cr.Vcluster.Name)

	return nil
}

func (cr *ClusterRegistration) FilterDeletedProjectPipelineItem() error {
	path := fmt.Sprintf("%s/%s/production/ingress-tekton-dashborard.yaml", concatHostClustesrDir(cr.TenantConfigRepoLocalPath), cr.Vcluster.HostCluster.Name)

	projectPipelineItems := make([]*ProjectPipelineItem, 0)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to get project pipeline infomation")
		}
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ingresses, err := parseIngresses(string(bytes))
	if err != nil {
		return err
	}
	for _, ingress := range ingresses {
		item := &ProjectPipelineItem{}
		re := regexp.MustCompile(`(.+?)-tekton-dashborard`)
		matches := re.FindStringSubmatch(ingress.Metadata.Name)
		if len(matches) == 0 {
			return fmt.Errorf("the ingress name %s may be modified and do not conform to specification", ingress.Metadata.Name)
		}
		item.Name = strings.TrimSpace(matches[1])

		item.HostClusterName = cr.Vcluster.HostCluster.Name

		item.TektonConfig = &TektonConfig{
			Host: ingress.Spec.Rules[0].Host,
			URL:  cr.Vcluster.HostCluster.ApiServer,
		}

		projectPipelineItems = append(projectPipelineItems, item)
	}

	for i, projectPipelineItem := range projectPipelineItems {
		if projectPipelineItem.Name == cr.Vcluster.Name {
			projectPipelineItems = append(projectPipelineItems[:i], projectPipelineItems[i+1:]...)
		}
	}

	cr.HostCluster.ProjectPipelineItems = projectPipelineItems

	return nil
}

func (cr *ClusterRegistration) DeleteRuntime(nodes *nodestree.Node) error {
	// Directly delete the specified Runtime file directory.
	runtimeDir := fmt.Sprintf("%s/%s", concatRuntimesDir(cr.TenantConfigRepoLocalPath), cr.Runtime.Name)
	err := DeleteSpecifyDir(runtimeDir)
	if err != nil {
		return err
	}

	// Replace the file node path read from the template configuration library.
	cr.ReplaceTemplatePathWithTenantRepositoryPath(nodes)

	// The virtual cluster needs to additionally delete the specified Vcluster directory.
	if IsVirtual(cr.Cluster) {
		vclustersDir := fmt.Sprintf("%s/%s", concatSpecifiedVclustersDir(cr.TenantConfigRepoLocalPath, cr.Vcluster.HostCluster.Name), cr.Cluster.Name)
		err := DeleteSpecifyDir(vclustersDir)
		if err != nil {
			return err
		}

		// Override template placeholders.
		OverlayTemplateDirectoryPlaceholder(nodes, _HostClusterDirectoreyPlaceholder, cr.Vcluster.HostCluster.Name)

		err = cr.FilterDeletedClusters()
		if err != nil {
			return err
		}
	}

	if cr.Usage == _VirtualProjectPipelineRuntime {
		cr.HostCluster.ProjectPipelineItems = DeleteProjectPipelineItems(cr.HostCluster.ProjectPipelineItems, cr.Cluster.Name)
	}

	err = cr.DeleteClusterToKustomization()
	if err != nil {
		return err
	}

	err = cr.Execute(nodes)
	if err != nil {
		return err
	}

	return nil
}

func DeleteProjectPipelineItems(itmes []*ProjectPipelineItem, clustrName string) []*ProjectPipelineItem {
	var result []*ProjectPipelineItem

	for _, item := range itmes {
		if item.Name != clustrName {
			result = append(result, item)
		}
	}

	return result
}

func (cr *ClusterRegistration) CleanupAppSetIfEmpty() error {
	switch {
	case cr.Usage == _HostCluster && len(cr.HostClusterNames) == 0:
		return cr.CleanHostClusterAppSet()
	case IsVirtual(cr.Cluster) && len(cr.VclusterNames) == 0:
		return cr.CleanVclusterAppSet()
	}

	return nil
}

func (cr *ClusterRegistration) CleanHostClusterAppSet() error {
	if cr.Usage != _HostCluster {
		return nil
	}

	hostClusterAppSetPath := fmt.Sprintf("%s/host-cluster-appset.yaml", concatTenantProductionDir(cr.TenantConfigRepoLocalPath))
	err := DeleteSpecifyFile(hostClusterAppSetPath)
	if err != nil {
		return err
	}

	return nil
}

func (cr *ClusterRegistration) DeleteRuntimeAppSet() error {
	if cr.Usage == _HostCluster {
		return nil
	}

	tenantProductionDir := concatTenantProductionDir(cr.TenantConfigRepoLocalPath)
	err := filepath.Walk(tenantProductionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && IsValidRuntimeAppSetFilename(info.Name()) {
			err := DeleteSpecifyFile(path)
			if err != nil {
				return fmt.Errorf("failed to delete file %s", info.Name())
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func IsValidRuntimeAppSetFilename(filename string) bool {
	pattern := "^runtime(-[a-zA-Z0-9]+)*-appset\\.yaml$"
	match, err := regexp.MatchString(pattern, filename)
	if err != nil {
		return false
	}
	return match
}

func (cr *ClusterRegistration) CleanVclusterAppSet() error {
	if !IsVirtual(cr.Cluster) {
		return nil
	}

	vclusterAppSetPath := fmt.Sprintf("%s/%s/%s", concatHostClustesrDir(cr.TenantConfigRepoLocalPath), cr.Vcluster.HostCluster.Name, _VclusterAppSetFile)
	err := DeleteSpecifyFile(vclusterAppSetPath)
	if err != nil {
		return err
	}

	return nil
}

func DeleteSpecifyDir(dir string) error {
	fileInfo, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("'%s' is not a directory, not allowed to delete", dir)
	}

	return os.RemoveAll(dir)
}

func DeleteSpecifyFile(filename string) error {
	fileInfo, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("'%s' is a directory, not allowed to delete", filename)
	}

	return os.Remove(filename)
}

func (cr *ClusterRegistration) DeleteClusterToNautes() error {
	filename := fmt.Sprintf("%s/%s.yaml", concatClustersDir(cr.TenantConfigRepoLocalPath), cr.Cluster.Name)
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	return os.Remove(filename)
}
