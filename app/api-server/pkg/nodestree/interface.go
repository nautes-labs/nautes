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

import "sigs.k8s.io/controller-runtime/pkg/client"

type NodesTree interface {
	Load(path string) (root Node, err error)
	AppendOperators(operator NodesOperator)
	Compare(options CompareOptions) error
	InsertNodes(nodes, resource *Node) (*Node, error)
	GetNode(nodes *Node, kind, name string) (node *Node)
	GetNodes() (*Node, error)
	RemoveNode(nodes *Node, node *Node) (*Node, error)
	FilterIgnoreByLayout(path string) error
}

type NodesOperator interface {
	CheckReference(options CompareOptions, node *Node, k8sClient client.Client) (bool, error)
	CreateResource(kind string) interface{}
	CreateNode(path string, data interface{}) (*Node, error)
	UpdateNode(node *Node, data interface{}) (*Node, error)
}
