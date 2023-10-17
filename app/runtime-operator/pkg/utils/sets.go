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

	"k8s.io/apimachinery/pkg/util/sets"
)

func NewStringSet(strs ...string) StringSet {
	return StringSet{
		Set: sets.New[string](strs...),
	}
}

type StringSet struct {
	sets.Set[string]
}

func (ss StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(ss.UnsortedList())
}

func (ss *StringSet) UnmarshalJSON(b []byte) error {
	var stringArray []string
	if err := json.Unmarshal(b, &stringArray); err != nil {
		return err
	}
	ss.Set = sets.New(stringArray...)
	return nil
}

func (ss StringSet) MarshalYAML() (interface{}, error) {
	return ss.UnsortedList(), nil
}

func (ss *StringSet) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var stringArray []string
	if err := unmarshal(&stringArray); err != nil {
		return err
	}
	ss.Set = sets.New(stringArray...)
	return nil
}

func (ss StringSet) Len() int {
	return len(ss.Set)
}
