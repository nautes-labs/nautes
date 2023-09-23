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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	resourcev1alpha1 "github.com/nautes-labs/nautes/api/kubernetes/v1alpha1"
	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	"sigs.k8s.io/yaml"

	kubernetes "github.com/nautes-labs/nautes/app/api-server/pkg/kubernetes"
	clusterconfig "github.com/nautes-labs/nautes/pkg/config/cluster"

	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
)

func NewClusterManagement(file FileOperation, nodestree nodestree.NodesTree) (ClusterRegistrationOperator, error) {
	clusterComponentConfig, err := clusterconfig.NewClusterComponentConfig()
	if err != nil {
		return nil, err
	}

	k8sClient, err := kubernetes.NewKubernetesOperator()
	if err != nil {
		return nil, err
	}

	return &ClusterManagement{
		clusterComponentConfig: clusterComponentConfig,
		file:                   file,
		nodestree:              nodestree,
		k8sClient:              k8sClient,
	}, nil
}

func (c *ClusterManagement) GetClsuter(tenantLocalPath, clusterName string) (*resourcev1alpha1.Cluster, error) {
	clusterFilePath := fmt.Sprintf("%s/%s.yaml", concatClustersDir(tenantLocalPath), clusterName)
	_, err := os.Stat(clusterFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("the cluster resource %s is not found", clusterName)
		}
		return nil, err
	}

	content, err := c.file.ReadFile(clusterFilePath)
	if err != nil {
		return nil, err
	}

	cluster := &resourcev1alpha1.Cluster{}
	err = yaml.Unmarshal(content, cluster)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (c *ClusterManagement) ListClusters(tenantLocalPath string) ([]*resourcev1alpha1.Cluster, error) {
	clustersDir := concatClustersDir(tenantLocalPath)
	files, err := c.file.ListFilesInDirectory(clustersDir)
	if err != nil {
		return nil, err
	}

	clusters := make([]*resourcev1alpha1.Cluster, 0)
	for _, filePath := range files {
		content, err := c.file.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		cluster := &resourcev1alpha1.Cluster{}
		err = yaml.Unmarshal(content, cluster)
		if err != nil {
			return nil, err
		}

		if cluster.Kind != "" &&
			cluster.Kind == nodestree.Cluster {
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

func (c *ClusterManagement) SaveCluster(params *ClusterRegistrationParams) error {
	var cluster = params.Cluster
	var clusters []resourcev1alpha1.Cluster
	var tenantRepoDir = params.Repo.TenantRepoDir
	var err error

	clusters, err = c.appendCluster(cluster, tenantRepoDir)
	if err != nil {
		return err
	}
	params.Clusters = clusters

	hostCluster, err := c.getHostCluster(clusters, cluster)
	if err != nil {
		return err
	}
	params.HostCluster = hostCluster

	installPaths, err := c.getInstallPaths(cluster, clusterconfig.ToSave, params.Repo)
	if err != nil {
		return err
	}

	clusterTemplateDir, err := c.generateTemplateFiles(installPaths, params.Repo)
	if err != nil {
		return err
	}

	defer c.file.DeleteDir(clusterTemplateDir)

	nodes, err := c.loadTemplateNodesTree(clusterTemplateDir)
	if err != nil {
		return err
	}

	if err := c.renderTemplate(nodes, params); err != nil {
		return err
	}

	if err := c.updateTenantConfiguration(nodes, cluster, params.HostCluster, params.Repo); err != nil {
		return err
	}

	if err := c.createClusterResource(cluster, tenantRepoDir); err != nil {
		return err
	}

	defer c.cleanExcessFiles(clusters, cluster, tenantRepoDir)

	if err = c.createDexCallback(params); err != nil {
		return err
	}

	return nil
}

func (c *ClusterManagement) RemoveCluster(params *ClusterRegistrationParams) error {
	var cluster = params.Cluster
	var clusters []resourcev1alpha1.Cluster
	var tenantRepoDir = params.Repo.TenantRepoDir
	var err error

	clusters, err = c.removeCluster(cluster, tenantRepoDir)
	if err != nil {
		return err
	}
	params.Clusters = clusters

	isDelete, err := c.isDeleteHostCluster(clusters, cluster)
	if !isDelete && err != nil {
		return err
	}

	hostCluster, err := c.getHostCluster(clusters, cluster)
	if err != nil {
		return err
	}
	params.HostCluster = hostCluster

	installPaths, err := c.getClusterCommonConfig(clusterconfig.ToRemove, &clusterconfig.ClusterInfo{
		Name:        cluster.Name,
		Usage:       string(cluster.Spec.Usage),
		ClusterType: string(cluster.Spec.ClusterType),
		WorkType:    string(cluster.Spec.WorkerType),
	}, params.Repo.ClusterTemplateDir)
	if err != nil {
		return err
	}

	clusterTemplateDir, err := c.generateTemplateFiles(installPaths, params.Repo)
	if err != nil {
		return err
	}

	defer c.file.DeleteDir(clusterTemplateDir)

	nodes, err := c.loadTemplateNodesTree(clusterTemplateDir)
	if err != nil {
		return err
	}

	if err := c.renderTemplate(nodes, params); err != nil {
		return err
	}

	if err := c.updateTenantConfiguration(nodes, cluster, params.HostCluster, params.Repo); err != nil {
		return err
	}

	if err = c.removeClusterResource(cluster, tenantRepoDir); err != nil {
		return err
	}

	if err := c.deleteSpecifyDir(cluster, params.HostCluster, tenantRepoDir); err != nil {
		return err
	}

	defer c.cleanExcessFiles(params.Clusters, cluster, tenantRepoDir)

	if err = c.removeDexCallback(params); err != nil {
		return err
	}

	return nil
}

// cleanExcessFiles It's a side effect function, Don't worry about mistakes.
func (c *ClusterManagement) cleanExcessFiles(clusters []resourcev1alpha1.Cluster, cluster *resourcev1alpha1.Cluster, tenantDir string) {
	if IsHostCluser(cluster) && isDeleteHostClusterAppSet(clusters) {
		hostClusterAppSetPath := fmt.Sprintf("%s/host-cluster-appset.yaml", concatTenantProductionDir(tenantDir))
		if err := c.file.DeleteFile(hostClusterAppSetPath); err != nil {
			return
		}
	}

	if IsHostCluser(cluster) && isDeleteVclusterClusterAppSet(clusters) {
		vclusterAppSetPath := fmt.Sprintf("%s/%s/%s", concatHostClustesrDir(tenantDir), cluster.Name, _VclusterAppSetFile)
		if err := c.file.DeleteFile(vclusterAppSetPath); err != nil {
			return
		}
	}
}

func isDeleteHostClusterAppSet(clusters []resourcev1alpha1.Cluster) bool {
	var isDelete = true
	for _, cluster := range clusters {
		if cluster.Spec.Usage == resourcev1alpha1.CLUSTER_USAGE_HOST {
			isDelete = false
		}
	}

	return isDelete
}

func isDeleteVclusterClusterAppSet(clusters []resourcev1alpha1.Cluster) bool {
	var isDelete = true
	for _, cluster := range clusters {
		if resourcev1alpha1.ClusterUsage(cluster.Spec.ClusterType) ==
			resourcev1alpha1.ClusterUsage(resourcev1alpha1.CLUSTER_TYPE_VIRTUAL) {
			isDelete = false
		}
	}

	return isDelete
}

func (c *ClusterManagement) getHostCluster(clusters []resourcev1alpha1.Cluster, cluster *resourcev1alpha1.Cluster) (*resourcev1alpha1.Cluster, error) {
	var hostCluster *resourcev1alpha1.Cluster
	var err error

	if IsVirtual(cluster) {
		hostCluster, err = c.getClusterByName(clusters, cluster.Spec.HostCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to get host cluster, err: %s", err)
		}
	}

	return hostCluster, nil
}

func (c *ClusterManagement) deleteSpecifyDir(cluster *resourcev1alpha1.Cluster, hostCluster *resourcev1alpha1.Cluster, tenantRepoDir string) error {
	var deletedDirs []string

	if IsHostCluser(cluster) {
		hostDir := fmt.Sprintf("%s/%s", concatHostClustesrDir(tenantRepoDir), cluster.Name)
		deletedDirs = append(deletedDirs, hostDir)
	} else {
		runtimeDir := fmt.Sprintf("%s/%s-runtime", concatRuntimesDir(tenantRepoDir), cluster.Name)
		deletedDirs = append(deletedDirs, runtimeDir)
	}

	if IsVirtual(cluster) {
		vclusterDir := fmt.Sprintf("%s/%s", concatSpecifiedVclustersDir(tenantRepoDir, hostCluster.Name), cluster.Name)
		deletedDirs = append(deletedDirs, vclusterDir)
	}

	return c.deleteDir(deletedDirs...)
}

func (c *ClusterManagement) deleteDir(dirs ...string) error {
	for _, dir := range dirs {
		if err := c.file.DeleteDir(dir); err != nil {
			return err
		}
	}

	return nil
}

func (c *ClusterManagement) isDeleteHostCluster(clusters []resourcev1alpha1.Cluster, cluster *resourcev1alpha1.Cluster) (bool, error) {
	if !IsHostCluser(cluster) {
		return true, nil
	}

	for _, c := range clusters {
		if c.Spec.HostCluster != "" &&
			c.Spec.HostCluster == cluster.Name {
			return false, fmt.Errorf("failed to remove cluster %s, this cluster is referenced by %s", cluster.Name, c.Name)
		}
	}
	return true, nil
}

func (c *ClusterManagement) createClusterResource(cluster *resourcev1alpha1.Cluster, tenantRepoDir string) error {
	bytes, err := json.Marshal(cluster)
	if err != nil {
		return err
	}

	bytes, err = yaml.JSONToYAML(bytes)
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/%s.yaml", concatClustersDir(tenantRepoDir), cluster.Name)
	return c.file.WriteFile(filePath, bytes)
}

// getComponentInstallPaths is a method that retrieves the installation paths of components in a cluster.
func (c *ClusterManagement) getComponentInstallPaths(cluster *resourcev1alpha1.Cluster, clusterTemplateDir string) ([]string, error) {
	var installPaths []string
	var componentsListMap = resourcev1alpha1.ConvertComponentsListToMap(cluster.Spec.ComponentsList)

	for _, component := range componentsListMap {
		if component == nil {
			continue
		}

		thirdPartComponent, err := c.clusterComponentConfig.GetThirdPartComponentByName(component.Name)
		if err != nil {
			return nil, err
		}

		paths := getComponentInstallPaths(thirdPartComponent, cluster)
		installPaths = append(installPaths, paths...)
	}

	for i := range installPaths {
		installPaths[i] = fmt.Sprintf("%s/%s", clusterTemplateDir, installPaths[i])
	}

	return installPaths, nil
}

func getComponentInstallPaths(thirdPartComponent *clusterconfig.ThridPartComponent, cluster *resourcev1alpha1.Cluster) []string {
	var paths []string

	if len(thirdPartComponent.InstallPath.Generic) > 0 {
		paths = append(paths, thirdPartComponent.InstallPath.Generic...)
	} else {
		key := fmt.Sprintf("%s&%s", cluster.Spec.Usage, cluster.Spec.ClusterType)
		paths = append(paths, thirdPartComponent.GetInstallPath(key)...)
	}

	return paths
}

func (c *ClusterManagement) getClusterByName(clusters []resourcev1alpha1.Cluster, name string) (*resourcev1alpha1.Cluster, error) {
	for _, cluster := range clusters {
		if cluster.Name == name {
			return &cluster, nil
		}
	}

	return nil, fmt.Errorf("cluster %s is not found", name)
}

func (c *ClusterManagement) getClusterCommonConfig(action clusterconfig.ClusterOperation, clusterInfo *clusterconfig.ClusterInfo, clusterTemplateDir string) ([]string, error) {
	paths, err := c.clusterComponentConfig.GetClusterCommonConfig(action, clusterInfo)
	if err != nil {
		return nil, err
	}

	for i := range paths {
		paths[i] = fmt.Sprintf("%s/%s", clusterTemplateDir, paths[i])
	}

	return paths, nil
}

func (c *ClusterManagement) getInstallPaths(cluster *resourcev1alpha1.Cluster, action clusterconfig.ClusterOperation, repo *RepositoriesInfo) ([]string, error) {
	componentsInstallPaths, err := c.getComponentInstallPaths(cluster, repo.ClusterTemplateDir)
	if err != nil {
		return nil, err
	}

	commonConfigInstallPaths, err := c.getClusterCommonConfig(action, &clusterconfig.ClusterInfo{
		Name:        cluster.Name,
		Usage:       string(cluster.Spec.Usage),
		ClusterType: string(cluster.Spec.ClusterType),
		WorkType:    string(cluster.Spec.WorkerType),
	}, repo.ClusterTemplateDir)
	if err != nil {
		return nil, err
	}

	installPaths := append(componentsInstallPaths, commonConfigInstallPaths...)

	return installPaths, nil
}

func (c *ClusterManagement) createTemporaryDirectory() (string, error) {
	clusterTemplateDir := fmt.Sprintf("%s/%s", os.TempDir(), ClusterTemplatesDir)
	dir, err := c.file.CreateDir(clusterTemplateDir)
	if err != nil {
		return "", err
	}

	c.setClusterTemplateDir(dir)

	return dir, nil
}

func (c *ClusterManagement) generateTemplateFiles(installPaths []string, repo *RepositoriesInfo) (string, error) {
	clusterTemplateDir, err := c.createTemporaryDirectory()
	if err != nil {
		return clusterTemplateDir, err
	}

	for _, filePath := range installPaths {
		isDir, err := c.file.IsDir(filePath)
		if err != nil {
			return clusterTemplateDir, err
		}

		if isDir {
			filePaths, err := c.file.ListFilesInDirectory(filePath)
			if err != nil {
				return clusterTemplateDir, err
			}
			for _, filePath := range filePaths {
				err = c.writeTemporaryFile(filePath, repo.ClusterTemplateDir, clusterTemplateDir)
				if err != nil {
					return clusterTemplateDir, err
				}
			}
		} else {
			err = c.writeTemporaryFile(filePath, repo.ClusterTemplateDir, clusterTemplateDir)
			if err != nil {
				return clusterTemplateDir, err
			}
		}
	}

	return clusterTemplateDir, nil
}

func (c *ClusterManagement) setClusterTemplateDir(dir string) {
	c.clusterTemplateDir = dir
}

func (c *ClusterManagement) getClusterTemplateDir() string {
	return c.clusterTemplateDir
}

func (c *ClusterManagement) writeTemporaryFile(filePath string, clusterTemplateDir, dir string) error {
	bytes, err := c.file.ReadFile(filePath)
	if err != nil {
		return err
	}

	path := utilstring.ReplacePath(filePath, clusterTemplateDir, dir)
	_, err = c.file.CreateFile(path)
	if err != nil {
		return err
	}

	err = c.file.WriteFile(path, bytes)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterManagement) appendCluster(cluster *resourcev1alpha1.Cluster, tenantRepoDir string) ([]resourcev1alpha1.Cluster, error) {
	clusters, err := c.getClusters(tenantRepoDir)
	if err != nil {
		return nil, err
	}

	if len(clusters) > 0 {
		clusters = addCluster(clusters, cluster)
	} else {
		clusters = append(clusters, *cluster)
	}

	return clusters, nil
}

func (c *ClusterManagement) removeCluster(cluster *resourcev1alpha1.Cluster, tenantRepoDir string) ([]resourcev1alpha1.Cluster, error) {
	clusters, err := c.getClusters(tenantRepoDir)
	if err != nil {
		return nil, err
	}

	for i, c := range clusters {
		if c.Name == cluster.Name {
			clusters = append(clusters[:i], clusters[i+1:]...)
		}
	}

	return clusters, nil
}

func (c *ClusterManagement) getClusters(tenantRepoDir string) ([]resourcev1alpha1.Cluster, error) {
	var clusters []resourcev1alpha1.Cluster

	storeClustersDir := fmt.Sprintf("%s/%s", tenantRepoDir, _NutesClusters)
	paths, err := c.file.ListFilesInDirectory(storeClustersDir)
	if err != nil {
		return nil, err
	}

	for _, filePath := range paths {
		bytes, err := c.file.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		cluster := resourcev1alpha1.Cluster{}
		if err = yaml.Unmarshal(bytes, &cluster); err == nil && cluster.Name != "" {
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

// createDexCallback Default to use dex as OAuth authentication service.
func (c *ClusterManagement) createDexCallback(parms *ClusterRegistrationParams) error {
	var cluster = parms.Cluster

	if !IsHostCluser(cluster) {
		url, err := c.GetDeploymentRedirectURI(parms)
		if err != nil {
			return err
		}

		err = c.AppendDexRedirectURIs(url)
		if err != nil {
			return err
		}
	}

	if IsPhysical(cluster) {
		url, err := c.GetOAuthProxyRedirectURI(parms)
		if err != nil {
			return err
		}

		err = c.AppendDexRedirectURIs(url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ClusterManagement) removeDexCallback(parms *ClusterRegistrationParams) error {
	var cluster = parms.Cluster

	if !IsHostCluser(cluster) {
		url, err := c.GetDeploymentRedirectURI(parms)
		if err != nil {
			return err
		}

		err = c.RemoveDexRedirectURIs(url)
		if err != nil {
			return err
		}
	}

	if IsPhysical(cluster) {
		url, err := c.GetOAuthProxyRedirectURI(parms)
		if err != nil {
			return err
		}

		err = c.RemoveDexRedirectURIs(url)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ClusterManagement) removeClusterResource(cluster *resourcev1alpha1.Cluster, tenantRepoDir string) error {
	dir := concatClustersDir(tenantRepoDir)
	filePath := fmt.Sprintf("%s/%s.yaml", dir, cluster.Name)
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}

	return c.file.DeleteFile(filePath)
}

func (c *ClusterManagement) updateTenantConfiguration(nodes *nodestree.Node, cluster *resourcev1alpha1.Cluster, hostCluster *resourcev1alpha1.Cluster, repo *RepositoriesInfo) error {
	err := c.refreshNodePath(cluster, nodes, repo, hostCluster)
	if err != nil {
		return err
	}

	if err := c.writeConfigByNodes(nodes); err != nil {
		return err
	}

	return nil
}

func (c *ClusterManagement) refreshNodePath(cluster *resourcev1alpha1.Cluster, nodes *nodestree.Node, repo *RepositoriesInfo, hostCluster *resourcev1alpha1.Cluster) error {
	if IsHostCluser(cluster) {
		re := replacePlaceholder{Placeholder: _HostClusterPlaceholder, Target: cluster.Name}
		if err := c.revisionFilePath(nodes, repo.TenantRepoDir, re); err != nil {
			return err
		}
	} else {
		re := []replacePlaceholder{{
			Placeholder: _RuntimePlaceholder, Target: fmt.Sprintf("%s-runtime", cluster.Name),
		}}
		if IsVirtual(cluster) {
			re = append(re, replacePlaceholder{Placeholder: _HostClusterPlaceholder, Target: hostCluster.Name})
			re = append(re, replacePlaceholder{Placeholder: _VclusterPlaceholder, Target: cluster.Name})
		}

		if err := c.revisionFilePath(nodes, repo.TenantRepoDir, re...); err != nil {
			return err
		}
	}
	return nil
}

// loadTemplateNodesTree Obtain the directory where the cluster template is located.
// Load the file nodes of this directory.
func (c *ClusterManagement) loadTemplateNodesTree(clusterTemplateDir string) (*nodestree.Node, error) {
	nodes, err := c.nodestree.Load(clusterTemplateDir)
	if err != nil {
		return nil, err
	}

	return &nodes, nil
}
func (c *ClusterManagement) writeConfigByNodes(nodes *nodestree.Node) error {
	for _, node := range nodes.Children {
		if node.IsDir {
			err := c.writeConfigByNodes(node)
			if err != nil {
				return err
			}
		} else {
			content, ok := node.Content.(string)
			if content == "" {
				content = fmt.Sprintf("# %s", node.Name)
			}
			if ok {
				err := c.file.WriteFile(node.Path, []byte(content))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type replacePlaceholder struct {
	Placeholder string
	Target      string
}

func (c *ClusterManagement) revisionFilePath(nodes *nodestree.Node, tenantRepoDir string, options ...replacePlaceholder) error {
	for _, re := range options {
		c.overlayTemplatePlaceholder(nodes, re.Placeholder, re.Target)
	}

	clusterTemplateDir := c.getClusterTemplateDir()
	c.replaceTemplateDirectory(nodes, clusterTemplateDir, tenantRepoDir)
	return nil
}

func (c *ClusterManagement) overlayTemplatePlaceholder(nodes *nodestree.Node, placeholder, replaceValue string) {
	stack := []*nodestree.Node{}
	stack = append(stack, nodes.Children...)

	for len(stack) > 0 {
		// Pop node from stack
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if strings.Contains(node.Path, placeholder) {
			if node.IsDir && node.Name == placeholder {
				node.Name = replaceValue
			}
			node.Path = strings.Replace(node.Path, placeholder, replaceValue, 1)
		}

		if node.IsDir {
			stack = append(stack, node.Children...)
		}
	}
}

func (c *ClusterManagement) replaceTemplateDirectory(nodes *nodestree.Node, target, current string) {
	for _, node := range nodes.Children {
		path := utilstring.ReplacePath(node.Path, target, current)
		node.Path = path
		if node.IsDir {
			c.replaceTemplateDirectory(node, target, current)
		}
	}
}

func (c *ClusterManagement) renderTemplate(nodes *nodestree.Node, params *ClusterRegistrationParams) error {
	for _, node := range nodes.Children {
		if node.IsDir {
			err := c.renderTemplate(node, params)
			if err != nil {
				return err
			}
		} else {
			content := node.Content.(string)
			tmpl := NewTemplate(params.Cluster.Name)
			if err := tmpl.Parse(content); err != nil {
				return err
			}

			buf := new(bytes.Buffer)
			if err := tmpl.Execute(buf, params); err != nil {
				return fmt.Errorf("failed to execute %s, err: %s", node.Path, err)
			}
			node.Content = buf.String()
		}
	}

	return nil
}

func (c *ClusterManagement) SaveClusterRuntime(nodes *nodestree.Node) error {
	return nil
}
