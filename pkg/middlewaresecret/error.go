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

package middlewaresecret

import (
	"errors"
	"fmt"
)

type MiddlewareSecretError struct {
	error
	ErrorType MiddlewareSecretErrorType
}

type MiddlewareSecretErrorType string

const (
	MiddlewareSecretErrorTypeNotFound           MiddlewareSecretErrorType = "NotFound"
	MiddlewareSecretErrorTypeFetched            MiddlewareSecretErrorType = "Fetched"
	MiddlewareSecretErrorTypeAlreadyExists      MiddlewareSecretErrorType = "AlreadyExists"
	MiddlewareSecretErrorTypeDisableIndexFailed MiddlewareSecretErrorType = "DisableIndexFailed"
)

func NewError(err error, errorType MiddlewareSecretErrorType) *MiddlewareSecretError {
	return &MiddlewareSecretError{
		error:     err,
		ErrorType: errorType,
	}
}

func ErrorNotFound(err error) *MiddlewareSecretError {
	return NewError(fmt.Errorf("secret not found: %w", err), MiddlewareSecretErrorTypeNotFound)
}

func IsNotFound(err error) bool {
	msErr := &MiddlewareSecretError{}
	if errors.As(err, msErr) {
		return msErr.ErrorType == MiddlewareSecretErrorTypeNotFound
	}
	return false
}

func ErrorFetched(err error) *MiddlewareSecretError {
	return NewError(fmt.Errorf("secret has been fetched: %w", err), MiddlewareSecretErrorTypeFetched)
}

func IsFetched(err error) bool {
	msErr := &MiddlewareSecretError{}
	if errors.As(err, msErr) {
		return msErr.ErrorType == MiddlewareSecretErrorTypeFetched
	}
	return false
}

func ErrorAlreadyExists(err error) *MiddlewareSecretError {
	return NewError(fmt.Errorf("secret already exists: %w", err), MiddlewareSecretErrorTypeAlreadyExists)
}

func IsAlreadyExists(err error) bool {
	msErr := &MiddlewareSecretError{}
	if errors.As(err, msErr) {
		return msErr.ErrorType == MiddlewareSecretErrorTypeAlreadyExists
	}
	return false
}

func ErrorDisableIndexFailed(err error) *MiddlewareSecretError {
	return NewError(fmt.Errorf("disable index failed: %w", err), MiddlewareSecretErrorTypeDisableIndexFailed)
}

func IsDisableIndexFailed(err error) bool {
	msErr := &MiddlewareSecretError{}
	if errors.As(err, msErr) {
		return msErr.ErrorType == MiddlewareSecretErrorTypeDisableIndexFailed
	}
	return false
}
