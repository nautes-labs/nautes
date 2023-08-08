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
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/nautes-labs/nautes/app/api-server/pkg/nodestree"
	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

// Save this function Create/Update cluster configuration based on cluster type
func (cr *ClusterRegistration) Save() error {
	nodes, err := cr.SaveClusterByType()
	if err != nil {
		return err
	}

	err = cr.SaveClusterConfig(nodes)
	if err != nil {
		return err
	}

	err = cr.SaveClusterToNautes()
	if err != nil {
		return err
	}

	return nil
}

func (cr *ClusterRegistration) SaveClusterByType() (*nodestree.Node, error) {
	var nodes nodestree.Node

	config, err := NewClusterFileIgnoreConfig(cr.ClusterTemplateRepoLocalPath)
	if err != nil {
		return nil, err
	}

	ignorePaths, ignoreFiles, err := cr.getSaveClusterIgnoreConfig(config)
	if err != nil {
		return nil, err
	}

	nodes, err = cr.LoadTemplateNodesTree(ignorePaths, ignoreFiles)
	if err != nil {
		return nil, err
	}

	err = cr.saveRuntimeByType(&nodes)
	if err != nil {
		return nil, err
	}

	return &nodes, nil
}

func (cr *ClusterRegistration) saveRuntimeByType(nodes *nodestree.Node) error {
	switch cr.Usage {
	case _HostCluster:
		return cr.SaveHostCluster(nodes)
	case _VirtualDeploymentRuntime:
		return cr.SaveRuntime(nodes)
	case _PhysicalDeploymentRuntime:
		return cr.SaveRuntime(nodes)
	case _VirtualProjectPipelineRuntime:
		return cr.SaveRuntime(nodes)
	case _PhysicalProjectPipelineRuntime:
		return cr.SaveRuntime(nodes)
	default:
		return errors.New("unknown cluster usage when saved runtime")
	}
}

func (cr *ClusterRegistration) getSaveClusterIgnoreConfig(config *ClusterFileIgnoreConfig) ([]string, []string, error) {
	switch cr.Usage {
	case _HostCluster:
		ignorePaths, ignoreFiles := config.GetSaveHostClusterConfig()
		return ignorePaths, ignoreFiles, nil
	case _PhysicalDeploymentRuntime:
		ignorePaths, ignoreFiles := config.GetSavePhysicalDeploymentRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _PhysicalProjectPipelineRuntime:
		ignorePaths, ignoreFiles := config.GetSavePhysicalProjectPipelineRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _VirtualDeploymentRuntime:
		ignorePaths, ignoreFiles := config.GetSaveVirtualDeploymentRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	case _VirtualProjectPipelineRuntime:
		ignorePaths, ignoreFiles := config.GetSaveVirtualProjectPipelineRuntimeConfig()
		return ignorePaths, ignoreFiles, nil
	default:
		return nil, nil, errors.New("unknown cluster usage")
	}
}

const (
	_ClusterKustomizationFile = "nautes/overlays/production/clusters/kustomization.yaml"
)

func (cr *ClusterRegistration) SaveClusterToKustomization() (err error) {
	kustomizationFilePath := fmt.Sprintf("%s/%s", cr.TenantConfigRepoLocalPath, _ClusterKustomizationFile)
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
		cr.ClusterResouceFiles = AddIfNotExists(kustomization.Resources, filename)
	} else {
		cr.ClusterResouceFiles = append(cr.ClusterResouceFiles, filename)
	}

	return nil
}

func (cr *ClusterRegistration) LoadTemplateNodesTree(ignorePath, ignoreFile []string) (nodes nodestree.Node, err error) {
	fileOptions := &nodestree.FileOptions{
		IgnorePath:       ignorePath,
		IgnoreFile:       ignoreFile,
		ExclusionsSuffix: []string{".txt", ".md"},
		ContentType:      nodestree.StringContentType,
	}
	client := nodestree.NewNodestree(fileOptions, nil, nil)
	nodes, err = client.Load(cr.ClusterTemplateRepoLocalPath)
	if err != nil {
		return
	}

	return
}

func (cr *ClusterRegistration) SaveHostCluster(nodes *nodestree.Node) error {
	err := cr.SaveClusterToKustomization()
	if err != nil {
		return err
	}

	err = cr.GetAndMergeHostClusterNames()
	if err != nil {
		return err
	}

	err = cr.Execute(nodes)
	if err != nil {
		return err
	}

	OverlayTemplateDirectoryPlaceholder(nodes, _HostClusterDirectoreyPlaceholder, cr.HostCluster.Name)

	cr.ReplaceTemplatePathWithTenantRepositoryPath(nodes)

	return nil
}

func (cr *ClusterRegistration) CheckHostClusterDirExists() error {
	if cr.Usage == _VirtualDeploymentRuntime && cr.Vcluster != nil {
		hostClusterDir := fmt.Sprintf("%s/%s", concatHostClustesrDir(cr.TenantConfigRepoLocalPath), cr.Vcluster.HostCluster.Name)
		_, err := os.Stat(hostClusterDir)
		if err != nil {
			return fmt.Errorf("the specified host cluster for this virtual cluster does not exist")
		}
	}

	return nil
}

func (cr *ClusterRegistration) SaveRuntime(nodes *nodestree.Node) error {
	err := cr.CheckHostClusterDirExists()
	if err != nil {
		return err
	}

	err = cr.SaveClusterToKustomization()
	if err != nil {
		return err
	}

	err = cr.GetAndMergeVclusterNames()
	if err != nil {
		return err
	}

	if cr.Usage == _VirtualProjectPipelineRuntime {
		cr.AddProjectPipelineItem()
	}

	err = cr.Execute(nodes)
	if err != nil {
		return err
	}

	if cr.Usage == _VirtualDeploymentRuntime || cr.Usage == _VirtualProjectPipelineRuntime {
		OverlayTemplateDirectoryPlaceholder(nodes, _VclusterDirectoryDirectoreyPlaceholder, cr.Vcluster.Name)
		OverlayTemplateDirectoryPlaceholder(nodes, _HostClusterDirectoreyPlaceholder, cr.Vcluster.HostCluster.Name)
	}

	OverlayTemplateDirectoryPlaceholder(nodes, _RuntimeDirectoryDirectoreyPlaceholder, cr.Runtime.Name)

	cr.ReplaceTemplatePathWithTenantRepositoryPath(nodes)

	return nil
}

func (cr *ClusterRegistration) AddProjectPipelineItem() {
	var tmpMap = make(map[string]bool, 0)
	for _, item := range cr.HostCluster.ProjectPipelineItems {
		tmpMap[item.Name] = true
	}
	if !tmpMap[cr.Vcluster.Name] {
		cr.HostCluster.ProjectPipelineItems = append(cr.HostCluster.ProjectPipelineItems, &ProjectPipelineItem{
			Name:            cr.Vcluster.Name,
			TektonConfig:    cr.Runtime.TektonConfig,
			HostClusterName: cr.Vcluster.HostCluster.Name,
		})
	}
}

func (cr *ClusterRegistration) GetAndMergeHostClusterNames() error {
	if cr.HostCluster == nil || cr.HostCluster.Name == "" {
		return nil
	}

	hostClusterAppsetFilePath := concatHostClusterAppsetFilePath(cr.TenantConfigRepoLocalPath)
	clusterNames, err := GetHostClusterNames(hostClusterAppsetFilePath)
	if err != nil {
		return err
	}
	cr.HostClusterNames = AddIfNotExists(clusterNames, cr.HostCluster.Name)

	return nil
}

func (cr *ClusterRegistration) GetAndMergeVclusterNames() error {
	if cr.Vcluster == nil {
		return nil
	}

	vclusterAppsetFilePath := fmt.Sprintf("%s/%s/%s/%s", cr.TenantConfigRepoLocalPath, _HostClustersDir, cr.Vcluster.HostCluster.Name, _VclusterAppSetFile)
	clusterNames, err := GetVclusterNames(vclusterAppsetFilePath)
	if err != nil {
		return err
	}

	cr.VclusterNames = AddIfNotExists(clusterNames, cr.Vcluster.Name)

	return nil
}

func (cr *ClusterRegistration) SaveClusterConfig(nodes *nodestree.Node) error {
	for _, node := range nodes.Children {
		if node.IsDir {
			err := cr.SaveClusterConfig(node)
			if err != nil {
				return err
			}
		} else {
			content, ok := node.Content.(string)
			if ok {
				err := WriteConfigFile(node.Path, content)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (cr *ClusterRegistration) SaveClusterToNautes() error {
	bytes, err := json.Marshal(cr.Cluster)
	if err != nil {
		return err
	}

	bytes, err = yaml.JSONToYAML(bytes)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s.yaml", concatClustersDir(cr.TenantConfigRepoLocalPath), cr.Cluster.Name)
	err = ioutil.WriteFile(filename, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func OverlayTemplateDirectoryPlaceholder(nodes *nodestree.Node, placeholder string, replaceValue string) {
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

func (cr *ClusterRegistration) ReplaceTemplatePathWithTenantRepositoryPath(nodes *nodestree.Node) {
	var oldDir, newDir string
	oldDir = cr.ClusterTemplateRepoLocalPath
	newDir = cr.TenantConfigRepoLocalPath

	for _, node := range nodes.Children {
		node.Path = ReplaceTemplatePathWithTenantRepositoryPath(node.Path, oldDir, newDir)
		if node.IsDir {
			cr.ReplaceTemplatePathWithTenantRepositoryPath(node)
		}
	}
}

func (cr *ClusterRegistration) Execute(nodes *nodestree.Node) error {
	for _, node := range nodes.Children {
		if node.IsDir {
			err := cr.Execute(node)
			if err != nil {
				return err
			}
		}

		if content, ok := node.Content.(string); ok {
			t, err := template.New(node.Name).Funcs(template.FuncMap{
				"split":          strings.Split,
				"randomString":   utilstring.RandStr,
				"getDomain":      utilstring.GetDomain,
				"encodeToString": utilstring.EncodeToString,
			}).Parse(content)
			if err != nil {
				return err
			}

			buf := new(bytes.Buffer)
			err = t.Execute(buf, cr)
			if err != nil {
				return fmt.Errorf("failed to execute %s, err: %s", node.Path, err)
			}
			node.Content = buf.String()
		}
	}

	return nil
}

func WriteConfigFile(filePath, content string) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReplaceTemplatePathWithTenantRepositoryPath(filePath, oldDir, newDir string) (newPath string) {
	return strings.Replace(filePath, oldDir, newDir, 1)
}
