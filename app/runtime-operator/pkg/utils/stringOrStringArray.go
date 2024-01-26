// Copyright 2024 Nautes Authors
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

type StringOrStringArray []string

func NewStringOrStringArray(input interface{}) StringOrStringArray {
	switch v := input.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	default:
		return []string{}
	}
}

func (ss *StringOrStringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	if err := unmarshal(&multi); err == nil {
		*ss = multi
		return nil
	}

	var single string
	if err := unmarshal(&single); err != nil {
		return err
	}
	*ss = []string{single}
	return nil
}
