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

import "fmt"

// IsValidResourceLayout testing the valid of resources
func CheckEffectiveResourceLayout(options CompareOptions, in *nodesTree) error {
	configInfos := listConfigInfos([]Config{*in.config})
	nodeInfos := listNodeInfos([]Node{options.Nodes}, configInfos)

	err := resourceEffectiveness(nodeInfos, configInfos)
	if err != nil {
		return err
	}

	return nil
}

// resourceEffectiveness checking the valid of resources, from the aspects of category, level and name
func resourceEffectiveness(nodeInfos []NodeInfo, configInfos []configInfo) error {
	isKindContained := false
	for _, node := range nodeInfos {
		for _, config := range configInfos {
			if node.Kind == config.Kind {
				err := verificationLayoutRules(node, config)
				if err != nil {
					return err
				}

				isKindContained = true
			}
		}

		if !isKindContained {
			return fmt.Errorf("this %s resource type is not allowed to exist", node.Name)
		}

		isKindContained = !isKindContained
	}
	return nil
}

func verificationLayoutRules(node NodeInfo, config configInfo) error {
	if node.Category != config.Category {
		return fmt.Errorf("this %s resource belongs to a directory that is not consistent with the template layout, expected is %s, but now is %s in %s", node.Kind, config.Category, node.Category, node.Path)
	}

	if node.Level != config.Level {
		return fmt.Errorf("this %s resource level is not consistent with the template layout, expected is %d, but now is %d in %s", node.Kind, config.Level, node.Level, node.Path)
	}

	if config.Name != "any" && config.Name != node.Name {
		return fmt.Errorf("the %s resource name is not consistent with the template layout, expected is %s, but now is %s in %s", node.Kind, config.Name, node.Name, node.Path)
	}

	return nil
}
