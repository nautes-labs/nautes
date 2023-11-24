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

package utils

import (
	"encoding/json"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/util/sets"
)

func GetStructHash(req interface{}) uint32 {
	reqStr, _ := json.Marshal(req)
	h := fnv.New32()
	_, _ = h.Write(reqStr)
	return h.Sum32()
}

func GetHashMap[T any](objs []T) (map[uint32]T, sets.Set[uint32]) {
	hashSet := sets.New[uint32]()
	hashMap := map[uint32]T{}
	for i := range objs {
		hash := GetStructHash(objs[i])
		if hashSet.Has(hash) {
			continue
		}

		hashSet.Insert(hash)
		hashMap[hash] = objs[i]
	}

	return hashMap, hashSet
}
