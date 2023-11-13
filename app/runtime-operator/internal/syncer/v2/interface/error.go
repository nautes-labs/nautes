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

package component

import (
	"errors"
	"fmt"
)

// ComponentError is the error type returned by the component.
type ComponentError struct {
	error
	reason ErrorReason
}

type ErrorReason string

const (
	ErrorReasonAccountNotFound ErrorReason = "AccountNotFound"
	ErrorReasonProductNotFound ErrorReason = "ProductNotFound"
)

func AccountNotFound(err error, name string) ComponentError {
	return ComponentError{
		error:  fmt.Errorf("account %s not found: %w", name, err),
		reason: ErrorReasonAccountNotFound,
	}
}

func IsAccountNotFound(err error) bool {
	var productErr ComponentError
	if errors.As(err, &productErr) {
		if productErr.reason == ErrorReasonAccountNotFound {
			return true
		}
	}
	return false
}

func ProductNotFound(err error, name string) ComponentError {
	return ComponentError{
		error:  fmt.Errorf("product %s not found: %w", name, err),
		reason: ErrorReasonAccountNotFound,
	}
}

func IsProductNotFound(err error) bool {
	var productErr ComponentError
	if errors.As(err, &productErr) {
		if productErr.reason == ErrorReasonProductNotFound {
			return true
		}
	}
	return false
}
