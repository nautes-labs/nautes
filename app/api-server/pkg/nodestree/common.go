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

package nodestree

type configInfo struct {
	Name     string
	Kind     string
	Category string
	Count    int
	Optional bool
	Level    int
}

// listConfigInfos Get resource info list according to index
func listConfigInfos(configs []Config) (infos []configInfo) {
	idx := len(configs) - 1
	for _, config := range configs[idx].Sub {
		if config.Kind == "" && config.Name != "any" {
			configs = append(configs, config)

			infos = append(infos, listConfigInfos(configs)...)
		} else if config.Kind == "" && config.Name == "any" {
			infos = append(infos, recurrenceConfig(config, configs[idx].Name)...)
		} else {
			infos = append(infos, configInfo{
				Name:     config.Name,
				Kind:     config.Kind,
				Category: configs[idx].Name,
				Count:    config.Count,
				Optional: config.Optional,
				Level:    config.Level,
			})
		}
	}

	return
}

func recurrenceConfig(config Config, category string) (infos []configInfo) {
	for _, child := range config.Sub {
		if child.Kind == "" {
			infos = append(infos, recurrenceConfig(child, category)...)
		} else {
			infos = append(infos, configInfo{
				Name:     child.Name,
				Kind:     child.Kind,
				Category: category,
				Count:    child.Count,
				Optional: child.Optional,
				Level:    child.Level,
			})
		}
	}

	return
}

type NodeInfo struct {
	Name     string
	Kind     string
	Path     string
	Category string
	Level    int
}

// listNodeInfos generate a complete node info list
func listNodeInfos(nodes []Node, configInfos []configInfo) (infos []NodeInfo) {
	idx := len(nodes) - 1
	for _, node := range nodes[idx].Children {
		ok := isTopLevelNode(configInfos, node.Name)
		if ok {
			nodes = append(nodes, *node)
			infos = append(infos, listNodeInfos(nodes, configInfos)...)
		} else if !ok && node.IsDir {
			infos = append(infos, recurrenceNode(*node, nodes[idx].Name)...)
		} else {
			infos = append(infos, NodeInfo{
				Name:     node.Name,
				Kind:     node.Kind,
				Path:     node.Path,
				Category: nodes[idx].Name,
				Level:    node.Level,
			})
		}
	}

	return
}

func isTopLevelNode(configInfos []configInfo, name string) bool {
	for _, config := range configInfos {
		if config.Category == name {
			return true
		}
	}

	return false
}

func recurrenceNode(node Node, category string) (infos []NodeInfo) {
	for _, node := range node.Children {
		if node.IsDir && node.Kind == "" {
			infos = append(infos, recurrenceNode(*node, category)...)
		} else {
			infos = append(infos, NodeInfo{
				Name:     node.Name,
				Kind:     node.Kind,
				Category: category,
				Path:     node.Path,
				Level:    node.Level,
			})
		}
	}

	return
}
