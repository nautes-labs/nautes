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

// IsResourceRepeatName Check whether the resource name is duplicate
func CheckResourceRepeatName(options CompareOptions, in *nodesTree) error {
	var node = options.Nodes
	var resourcesLength = len(node.Children)

	for i := 0; i < resourcesLength; i++ {
		node1 := node.Children[i]
		if !node.Children[i].IsDir {
			for j := i + 1; j < resourcesLength; j++ {
				node2 := node.Children[j]
				err := compareNodeName(node1, node2)
				if err != nil {
					return err
				}
			}
		} else {
			child := CompareOptions{
				Nodes:       *node1,
				ProductName: options.ProductName,
			}
			err := CheckResourceRepeatName(child, in)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
