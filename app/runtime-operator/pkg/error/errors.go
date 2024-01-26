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

package errors

import (
	"errors"
	"fmt"
)

type ErrorReason string

const (
	ResourceNotBelongsToProduct ErrorReason = "resource not belongs to product"
	TransformRuleNotFound       ErrorReason = "transform rule not found"
	TransformRuleExists         ErrorReason = "transform rule exists"
	ResourceResourceNotFound    ErrorReason = "resource not found"
	ResourceResourceExists      ErrorReason = "resource exists"
)

type runtimeErrors struct {
	reason ErrorReason
	error
}

func ErrorResourceNotBelongsToProduct(resourceName, product string) error {
	return runtimeErrors{
		reason: ResourceNotBelongsToProduct,
		error:  fmt.Errorf("resource %s is not belongs to product %s", resourceName, product),
	}
}

func IsErrorNotBelongsToProduct(err error) bool {
	runtimeErr := &runtimeErrors{}
	if errors.As(err, runtimeErr) {
		return runtimeErr.reason == ResourceNotBelongsToProduct
	}
	return false
}

func ErrorResourceNotFound(err error) error {
	return runtimeErrors{
		reason: ResourceResourceNotFound,
		error:  fmt.Errorf("resource not found: %w", err),
	}
}

func IsResourceNotFoundError(err error) bool {
	runtimeErr := &runtimeErrors{}
	if errors.As(err, runtimeErr) {
		return runtimeErr.reason == ResourceResourceNotFound
	}
	return false
}

func ErrorResourceExists(err error) error {
	return runtimeErrors{
		reason: ResourceResourceExists,
		error:  fmt.Errorf("resource exists: %w", err),
	}
}

func IsResourceExistsError(err error) bool {
	runtimeErr := &runtimeErrors{}
	if errors.As(err, runtimeErr) {
		return runtimeErr.reason == ResourceResourceExists
	}
	return false
}

func ErrorTransformRuleNotFound(err error) error {
	return runtimeErrors{
		reason: TransformRuleNotFound,
		error:  fmt.Errorf("transform rule not found: %w", err),
	}
}

func IsTransformRuleNotFoundError(err error) bool {
	runtimeErr := &runtimeErrors{}
	if errors.As(err, runtimeErr) {
		return runtimeErr.reason == TransformRuleNotFound
	}
	return false
}

func ErrorTransformRuleExists(err error) error {
	return runtimeErrors{
		reason: TransformRuleExists,
		error:  fmt.Errorf("transform rule exists: %w", err),
	}
}
