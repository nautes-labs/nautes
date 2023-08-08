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
	"strings"

	"reflect"
)

type StringSelector struct {
	field     string
	matchMode SelectorOperatorSymbol
}

// TODO
// 增加operator操作符的入参
func NewStringSelector(field string, matchMode SelectorOperatorSymbol) *StringSelector {
	return &StringSelector{field: field, matchMode: matchMode}
}

func (s *StringSelector) GetField() string {
	return s.field
}

func (s *StringSelector) GetMatchFunc(operatorSymbol string) (MatchFunc, error) {
	switch {
	case operatorSymbol == EqualOperator && s.matchMode == Eq:
		return s.Eq, nil
	case operatorSymbol == NotEqualOperator && s.matchMode == NotEq:
		return s.NotEq, nil
	case operatorSymbol == EqualOperator && s.matchMode == In:
		return s.In, nil
	case operatorSymbol == NotEqualOperator && s.matchMode == NotIn:
		return s.NotIn, nil
	}

	return nil, fmt.Errorf("GetMatchFunc: no matching method, symbol: %s, match mode: %s", operatorSymbol, s.matchMode)
}

func (s *StringSelector) Eq(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkStringType(value)
	if err != nil {
		return false, err
	}

	return value.String() == matchValue, nil
}

func (s *StringSelector) NotEq(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkStringType(value)
	if err != nil {
		return false, err
	}

	return value.String() != matchValue, nil
}

func (s *StringSelector) In(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkStringType(value)
	if err != nil {
		return false, err
	}

	return strings.Contains(value.String(), matchValue), nil
}

func (s *StringSelector) NotIn(value reflect.Value, matchValue string) (bool, error) {
	err := s.checkStringType(value)
	if err != nil {
		return false, err
	}

	return !strings.Contains(value.String(), matchValue), nil
}

func (s *StringSelector) checkStringType(value reflect.Value) error {
	t := reflect.TypeOf(value.Interface())
	if t.Kind() != reflect.String {
		return fmt.Errorf("inconsistent data type, expected is 'String', but now is '%v'", strings.Title(t.Kind().String()))
	}

	return nil
}
