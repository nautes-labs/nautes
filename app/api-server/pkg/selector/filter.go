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
	"regexp"
	"strings"
)

type Filter struct {
	Field  string
	Value  string
	Symbol string
	MatchFunc
}

func ParseFilterFieldString(fieldSelectorStr string) (filters []*Filter, err error) {
	splits := strings.Split(fieldSelectorStr, ",")

	for _, split := range splits {
		symbol := findSymbol(split)
		if symbol == "" {
			return nil, fmt.Errorf("this operation symbol is not supported: %s", split)
		}

		keyValue := strings.Split(split, symbol)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("the parameter %s is wrong, eg: key=value", fieldSelectorStr)
		}

		filters = append(filters, &Filter{
			Field:  keyValue[0],
			Value:  keyValue[1],
			Symbol: symbol,
		})
	}

	return
}

func findSymbol(str string) string {
	re := regexp.MustCompile(`[^.\w\s]+`)
	match := re.FindString(str)

	if match != "" {
		return match
	} else {
		return ""
	}
}
