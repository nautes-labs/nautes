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

package selector

import (
	"fmt"
	"reflect"
	"strings"
)

func Match(fieldSelectorStr string, content interface{}, rules map[string]map[string]FieldSelector) (bool, error) {
	if fieldSelectorStr == "" {
		return true, nil
	}

	filters, err := ParseFilterFieldString(fieldSelectorStr)
	if err != nil {
		return false, err
	}

	for _, filter := range filters {
		selector, ok := rules[filter.Field][filter.Symbol]
		if !ok {
			return false, fmt.Errorf("failed to get selector, the filter field %s operator symbol %s is not supported", filter.Field, filter.Symbol)
		}

		matchFn, err := selector.GetMatchFunc(filter.Symbol)
		if err != nil {
			return false, err
		}
		filter.MatchFunc = matchFn
		filter.Field = selector.GetField()

		matches, err := matchResource(content, filter)
		if err != nil {
			return false, fmt.Errorf("failed to match resource, err: %w", err)
		}
		if !matches {
			return false, nil
		}
	}

	return true, nil
}

func matchResource(entity interface{}, filter *Filter) (bool, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return false, fmt.Errorf("the entity must be a non-nil pointer")
	}
	return matchValue(val.Elem(), filter.Value, filter.Field, filter.MatchFunc)
}

func matchValue(f reflect.Value, value, field string, matchFunc MatchFunc) (bool, error) {
	fieldNames := strings.Split(field, ".")

	for i, fieldName := range fieldNames {
		f = f.FieldByName(fieldName)

		if !f.IsValid() {
			return false, fmt.Errorf("the field %s value is invalid", fieldName)
		}

		// If the 'In' operator is included, the value matching of the array elements is performed.
		t := reflect.TypeOf(f.Interface())
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			for j := 0; j < f.Len(); j++ {
				v := f.Index(j)

				// If the field include "In" symbol, then will compare item
				if i+1 == len(fieldNames)-1 && fieldNames[i+1] == string(In) {
					match, err := matchFunc(v, value)
					if err != nil {
						return false, err
					}
					if match {
						return true, nil
					}

					continue
				}

				// recursively call matchValue for each element in the array/slice
				match, err := matchValue(v, value, strings.Join(fieldNames[i+1:], "."), matchFunc)
				if err != nil {
					return false, err
				}
				if match {
					return true, nil
				}
			}
			break
		}

		// if this is the last field, run the matching function on the value
		if i == len(fieldNames)-1 {
			return matchFunc(f, value)
		}

		// if the current field is a struct or pointer to a struct, continue to the next field
		if t.Kind() == reflect.Struct {
			continue
		} else if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
			f = f.Elem()
			continue
		}

		// if the current field is not a struct or pointer to a struct and there are more fields to check,
		// return an error since we cannot continue any further
		return false, fmt.Errorf("matchResource: field %s is not a struct or pointer to struct", fieldName)
	}

	return false, nil
}
