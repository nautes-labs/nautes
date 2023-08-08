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
	"strconv"
	"strings"
)

type BoolSelector struct {
	field     string
	matchMode SelectorOperatorSymbol
}

func NewBoolSelector(field string, matchMode SelectorOperatorSymbol) *BoolSelector {
	return &BoolSelector{field: field, matchMode: matchMode}
}

func (s *BoolSelector) GetField() string {
	return s.field
}

func (s *BoolSelector) GetMatchFunc(operatorSymbol string) (MatchFunc, error) {
	switch {
	case operatorSymbol == EqualOperator:
		return s.Eq, nil
	case operatorSymbol == NotEqualOperator:
		return s.NotEq, nil
	}

	return nil, fmt.Errorf("GetMatchFunc: no matching method, symbol: %s", operatorSymbol)
}

func (s *BoolSelector) Eq(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkBoolType(value)
	if err != nil {
		return false, err
	}

	boolVal, err := strconv.ParseBool(matchValue)
	if err != nil {
		return false, fmt.Errorf("match value must be a Boolean value: %s", matchValue)
	}

	return value.Bool() == boolVal, nil
}

func (s *BoolSelector) NotEq(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkBoolType(value)
	if err != nil {
		return false, err
	}

	boolVal, err := strconv.ParseBool(matchValue)
	if err != nil {
		return false, fmt.Errorf("match value must be a Boolean value: %s", matchValue)
	}

	return !value.Bool() == boolVal, nil
}

func (s *BoolSelector) checkBoolType(value reflect.Value) error {
	t := reflect.TypeOf(value.Interface())
	if t.Kind() != reflect.Bool {
		return fmt.Errorf("inconsistent data type, expected is 'Bool', but now is '%v'", strings.Title(t.Kind().String()))
	}

	return nil
}
