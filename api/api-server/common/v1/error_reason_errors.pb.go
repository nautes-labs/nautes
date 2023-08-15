// Code generated by protoc-gen-go-errors. DO NOT EDIT.

package v1

import (
	fmt "fmt"
	errors "github.com/go-kratos/kratos/v2/errors"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
const _ = errors.SupportPackageIsVersion1

func IsProjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_PROJECT_NOT_FOUND.String() && e.Code == 404
}

func ErrorProjectNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_PROJECT_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsGroupNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_GROUP_NOT_FOUND.String() && e.Code == 404
}

func ErrorGroupNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_GROUP_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsNodeNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_NODE_NOT_FOUND.String() && e.Code == 404
}

func ErrorNodeNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_NODE_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_RESOURCE_NOT_FOUND.String() && e.Code == 404
}

func ErrorResourceNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_RESOURCE_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsResourceNotMatch(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_RESOURCE_NOT_MATCH.String() && e.Code == 500
}

func ErrorResourceNotMatch(format string, args ...interface{}) *errors.Error {
	return errors.New(500, ErrorReason_RESOURCE_NOT_MATCH.String(), fmt.Sprintf(format, args...))
}

func IsNoAuthorization(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_NO_AUTHORIZATION.String() && e.Code == 403
}

func ErrorNoAuthorization(format string, args ...interface{}) *errors.Error {
	return errors.New(403, ErrorReason_NO_AUTHORIZATION.String(), fmt.Sprintf(format, args...))
}

func IsDeploykeyNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_DEPLOYKEY_NOT_FOUND.String() && e.Code == 404
}

func ErrorDeploykeyNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_DEPLOYKEY_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsSecretNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_SECRET_NOT_FOUND.String() && e.Code == 404
}

func ErrorSecretNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_SECRET_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsAccesstokenNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_ACCESSTOKEN_NOT_FOUND.String() && e.Code == 404
}

func ErrorAccesstokenNotFound(format string, args ...interface{}) *errors.Error {
	return errors.New(404, ErrorReason_ACCESSTOKEN_NOT_FOUND.String(), fmt.Sprintf(format, args...))
}

func IsRefreshPermissionsAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == ErrorReason_REFRESH_PERMISSIONS_ACCESS_DENIED.String() && e.Code == 403
}

func ErrorRefreshPermissionsAccessDenied(format string, args ...interface{}) *errors.Error {
	return errors.New(403, ErrorReason_REFRESH_PERMISSIONS_ACCESS_DENIED.String(), fmt.Sprintf(format, args...))
}