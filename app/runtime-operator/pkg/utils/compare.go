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
