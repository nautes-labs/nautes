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

import (
	"fmt"
	"path/filepath"

	utilstring "github.com/nautes-labs/nautes/app/api-server/util/string"
)

const (
	ResourceNameCluster = "clusters"
)

var (
	gitlab = "gitlab"
)

// CheckResouceReference Detect resource references
func CheckResouceReference(options CompareOptions, in *nodesTree) error {
	mapping := make(map[string]*Node)
	NodesToMapping(&options.Nodes, mapping)

	for _, node := range mapping {
		if !node.IsDir {
			for _, o := range in.operators {
				ok, err := o.CheckReference(options, node, in.client)
				if ok && err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func CheckResourceSubdirectory(nodes, node *Node) error {
	subDir := filepath.Dir(node.Path)
	mapping := make(map[string]*Node)
	NodesToMapping(nodes, mapping)
	subNode, ok := mapping[subDir]
	if !ok {
		return fmt.Errorf("the file path node is not found: %s", subDir)
	}

	nodeName := GetResourceValue(node.Content, "ObjectMeta", "Name")
	if subNode.Name != nodeName {
		return fmt.Errorf("the resource directory name is inconsistent, the subdirectory name of %s resource should be %s, but now it is %s", nodeName, nodeName, subNode.Name)
	}

	return nil
}

func IsResourceExist(options CompareOptions, targetResourceName, resourceKind string) bool {
	resourceNodes := ListsResourceNodes(options.Nodes, resourceKind)
	for _, node := range resourceNodes {
		if node.Kind == resourceKind {
			kind := GetResourceValue(node.Content, "TypeMeta", "Kind")
			if kind != resourceKind {
				return false
			}
			if kind == resourceKind {
				name := GetResourceValue(node.Content, "ObjectMeta", "Name")
				if name == targetResourceName {
					return true
				}
			}
		}
	}

	return false
}

var (
	gitlabWebhook = []string{
		"confidential_issues_events",
		"confidential_note_events",
		"deployment_events",
		"issues_events",
		"job_events",
		"merge_requests_events",
		"note_events",
		"pipeline_events",
		"push_events",
		"releases_events",
		"tag_push_events",
	}
)

func CheckGitHooks(gitType string, events []string) error {
	switch gitType {
	case gitlab:
		for _, event := range events {
			if event == "" {
				continue
			}

			if ok := utilstring.ContainsString(gitlabWebhook, event); !ok {
				return fmt.Errorf("invalid webhook %s please refer to gitlab api documentation: https://docs.gitlab.com/ee/api/projects.html#edit-project-hook", event)
			}
		}
	default:
		return fmt.Errorf("'%s' platform not supported", gitType)
	}

	return nil
}

func compareNodeName(node1, node2 *Node) error {
	if node2.IsDir {
		return nil
	}

	kind1 := GetResourceValue(node1.Content, "TypeMeta", "Kind")
	name1 := GetResourceValue(node1.Content, "ObjectMeta", "Name")
	kind2 := GetResourceValue(node2.Content, "TypeMeta", "Kind")
	name2 := GetResourceValue(node2.Content, "ObjectMeta", "Name")

	if kind1 == kind2 && name1 == name2 {
		return fmt.Errorf("duplicate names are not allowed for resources, duplicate files are %s and %s ", node1.Name, node2.Name)
	}

	return nil
}
