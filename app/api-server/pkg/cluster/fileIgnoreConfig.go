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
	"fmt"
	"io/ioutil"
	"os"

	yaml "sigs.k8s.io/yaml"
)

const (
	ClusterfilterFileName = "clusterignorerule"
)

type ClusterFileIgnoreConfig struct {
	Save   Save         `yaml:"save"`
	Remove Remove       `yaml:"remove"`
	Common CommonConfig `yaml:"common"`
}

type Save struct {
	HostCluster                    HostClusterConfig `yaml:"hostCluster"`
	PhysicalDeploymentRuntime      RuntimeConfig     `yaml:"physicalDeploymentRuntime"`
	PhysicalProjectPipelineRuntime RuntimeConfig     `yaml:"physicalProjectPipelineRuntime"`
	VirtualDeploymentRuntime       RuntimeConfig     `yaml:"virtualDeploymentRuntime"`
	VirtualProjectPipelineRuntime  RuntimeConfig     `yaml:"virtualDeploymentRuntime"`
}

type Remove struct {
	HostCluster                    HostClusterConfig `yaml:"hostCluster"`
	PhysicalDeploymentRuntime      RuntimeConfig     `yaml:"physicalDeploymentRuntime"`
	PhysicalProjectPipelineRuntime RuntimeConfig     `yaml:"physicalProjectPipelineRuntime"`
	VirtualDeploymentRuntime       RuntimeConfig     `yaml:"virtualDeploymentRuntime"`
	VirtualProjectPipelineRuntime  RuntimeConfig     `yaml:"virtualDeploymentRuntime"`
}

type HostClusterConfig struct {
	IgnorePath []string `yaml:"ignorePath"`
	IgnoreFile []string `yaml:"ignoreFile"`
}

type RuntimeConfig struct {
	IgnorePath []string `yaml:"ignorePath"`
	IgnoreFile []string `yaml:"ignoreFile"`
}

type CommonConfig struct {
	IgnorePath []string `yaml:"ignorePath"`
	IgnoreFile []string `yaml:"ignoreFile"`
}

func NewClusterFileIgnoreConfig(dir string) (*ClusterFileIgnoreConfig, error) {
	file := fmt.Sprintf("%s/%s.yaml", dir, ClusterfilterFileName)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var config *ClusterFileIgnoreConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (c *ClusterFileIgnoreConfig) GetSaveHostClusterConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Save.HostCluster.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Save.HostCluster.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetSavePhysicalDeploymentRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Save.PhysicalDeploymentRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Save.PhysicalDeploymentRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetSavePhysicalProjectPipelineRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Save.PhysicalProjectPipelineRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Save.PhysicalProjectPipelineRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetSaveVirtualDeploymentRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Save.VirtualDeploymentRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Save.VirtualDeploymentRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetSaveVirtualProjectPipelineRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Save.VirtualProjectPipelineRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Save.VirtualProjectPipelineRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetRemoveVirtualProjectPipelineRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Remove.VirtualProjectPipelineRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Remove.VirtualProjectPipelineRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetRemoveHostClusterConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Remove.HostCluster.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Remove.HostCluster.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetRemovePhysicalDeploymentRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Remove.PhysicalDeploymentRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Remove.PhysicalDeploymentRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetRemovePhysicalProjectPipelineRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Remove.PhysicalProjectPipelineRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Remove.PhysicalProjectPipelineRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}

func (c *ClusterFileIgnoreConfig) GetRemoveVirtualDeploymentRuntimeConfig() (ignorePath, ignoreFile []string) {
	ignorePath = append(ignorePath, c.Remove.VirtualDeploymentRuntime.IgnorePath...)
	ignorePath = append(ignorePath, c.Common.IgnorePath...)
	ignoreFile = append(ignoreFile, c.Remove.VirtualDeploymentRuntime.IgnoreFile...)
	ignoreFile = append(ignoreFile, c.Common.IgnoreFile...)

	return
}
