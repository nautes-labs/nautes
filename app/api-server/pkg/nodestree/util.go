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
	"os"
	"reflect"
	"strings"

	structs "github.com/fatih/structs"
)

func NodesToMapping(nodes *Node, mapping map[string]*Node) {
	mapping[nodes.Path] = nodes
	for _, n := range nodes.Children {
		if n.IsDir {
			NodesToMapping(n, mapping)
		}

		mapping[n.Path] = n
	}
}

func IsInSlice(slice []string, s string) (isIn bool) {
	if len(slice) == 0 {
		return false
	}

	isIn = false
	for _, f := range slice {
		if f == s {
			isIn = true
			break
		}
	}

	return
}

func InContainsDir(str string, fields []string) (isIn bool) {
	ok := isFilePath(str)
	if ok {
		return false
	}
	isIn = false
	for _, f := range fields {
		if strings.Contains(str, f) {
			isIn = true
			break
		}
	}

	return
}

func isFilePath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}

	return !info.IsDir()
}

func GetResourceValue(c interface{}, field, key string) string {
	t := reflect.TypeOf(c)
	t = t.Elem()
	v := reflect.ValueOf(c)
	v = v.Elem()

	for i := 0; i < v.NumField(); i++ {
		if t.Field(i).Name == field {
			m := structs.Map(v.Field(i).Interface())
			v := m[key]

			if v != nil {
				return v.(string)
			}
		}
	}

	return ""
}

type NodeFilterOptions func(node *Node) bool

func ListsResourceNodes(nodes Node, kind string, options ...NodeFilterOptions) (list []*Node) {
	for _, node := range nodes.Children {
		if node.IsDir {
			list = append(list, ListsResourceNodes(*node, kind, options...)...)
		} else {
			if node.Kind == kind {
				isAppend := true
				for _, fn := range options {
					if !fn(node) {
						isAppend = false
						break
					}
				}
				if isAppend {
					list = append(list, node)
				}
			}
		}
	}

	return list
}
