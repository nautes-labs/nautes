// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: deploymentruntime/v1/deploymentruntime.proto

package v1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on ManifestSource with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *ManifestSource) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ManifestSource with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ManifestSourceMultiError,
// or nil if none found.
func (m *ManifestSource) ValidateAll() error {
	return m.validate(true)
}

func (m *ManifestSource) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if utf8.RuneCountInString(m.GetCodeRepo()) < 1 {
		err := ManifestSourceValidationError{
			field:  "CodeRepo",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetTargetRevision()) < 1 {
		err := ManifestSourceValidationError{
			field:  "TargetRevision",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if utf8.RuneCountInString(m.GetPath()) < 1 {
		err := ManifestSourceValidationError{
			field:  "Path",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return ManifestSourceMultiError(errors)
	}

	return nil
}

// ManifestSourceMultiError is an error wrapping multiple validation errors
// returned by ManifestSource.ValidateAll() if the designated constraints
// aren't met.
type ManifestSourceMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ManifestSourceMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ManifestSourceMultiError) AllErrors() []error { return m }

// ManifestSourceValidationError is the validation error returned by
// ManifestSource.Validate if the designated constraints aren't met.
type ManifestSourceValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ManifestSourceValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ManifestSourceValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ManifestSourceValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ManifestSourceValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ManifestSourceValidationError) ErrorName() string { return "ManifestSourceValidationError" }

// Error satisfies the builtin error interface
func (e ManifestSourceValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sManifestSource.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ManifestSourceValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ManifestSourceValidationError{}

// Validate checks the field values on GetRequest with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *GetRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in GetRequestMultiError, or
// nil if none found.
func (m *GetRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *GetRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ProductName

	// no validation rules for DeploymentruntimeName

	if len(errors) > 0 {
		return GetRequestMultiError(errors)
	}

	return nil
}

// GetRequestMultiError is an error wrapping multiple validation errors
// returned by GetRequest.ValidateAll() if the designated constraints aren't met.
type GetRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetRequestMultiError) AllErrors() []error { return m }

// GetRequestValidationError is the validation error returned by
// GetRequest.Validate if the designated constraints aren't met.
type GetRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetRequestValidationError) ErrorName() string { return "GetRequestValidationError" }

// Error satisfies the builtin error interface
func (e GetRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetRequestValidationError{}

// Validate checks the field values on GetReply with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *GetReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GetReply with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in GetReplyMultiError, or nil
// if none found.
func (m *GetReply) ValidateAll() error {
	return m.validate(true)
}

func (m *GetReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Product

	// no validation rules for Name

	if all {
		switch v := interface{}(m.GetManifestSource()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, GetReplyValidationError{
					field:  "ManifestSource",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, GetReplyValidationError{
					field:  "ManifestSource",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetManifestSource()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return GetReplyValidationError{
				field:  "ManifestSource",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetDestination()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, GetReplyValidationError{
					field:  "Destination",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, GetReplyValidationError{
					field:  "Destination",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetDestination()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return GetReplyValidationError{
				field:  "Destination",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return GetReplyMultiError(errors)
	}

	return nil
}

// GetReplyMultiError is an error wrapping multiple validation errors returned
// by GetReply.ValidateAll() if the designated constraints aren't met.
type GetReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GetReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GetReplyMultiError) AllErrors() []error { return m }

// GetReplyValidationError is the validation error returned by
// GetReply.Validate if the designated constraints aren't met.
type GetReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GetReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GetReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GetReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GetReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GetReplyValidationError) ErrorName() string { return "GetReplyValidationError" }

// Error satisfies the builtin error interface
func (e GetReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGetReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GetReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GetReplyValidationError{}

// Validate checks the field values on ListsRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *ListsRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListsRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ListsRequestMultiError, or
// nil if none found.
func (m *ListsRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *ListsRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ProductName

	// no validation rules for FieldSelector

	if len(errors) > 0 {
		return ListsRequestMultiError(errors)
	}

	return nil
}

// ListsRequestMultiError is an error wrapping multiple validation errors
// returned by ListsRequest.ValidateAll() if the designated constraints aren't met.
type ListsRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListsRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListsRequestMultiError) AllErrors() []error { return m }

// ListsRequestValidationError is the validation error returned by
// ListsRequest.Validate if the designated constraints aren't met.
type ListsRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListsRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListsRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListsRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListsRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListsRequestValidationError) ErrorName() string { return "ListsRequestValidationError" }

// Error satisfies the builtin error interface
func (e ListsRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListsRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListsRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListsRequestValidationError{}

// Validate checks the field values on ListsReply with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *ListsReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ListsReply with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in ListsReplyMultiError, or
// nil if none found.
func (m *ListsReply) ValidateAll() error {
	return m.validate(true)
}

func (m *ListsReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	for idx, item := range m.GetItems() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ListsReplyValidationError{
						field:  fmt.Sprintf("Items[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ListsReplyValidationError{
						field:  fmt.Sprintf("Items[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ListsReplyValidationError{
					field:  fmt.Sprintf("Items[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ListsReplyMultiError(errors)
	}

	return nil
}

// ListsReplyMultiError is an error wrapping multiple validation errors
// returned by ListsReply.ValidateAll() if the designated constraints aren't met.
type ListsReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ListsReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ListsReplyMultiError) AllErrors() []error { return m }

// ListsReplyValidationError is the validation error returned by
// ListsReply.Validate if the designated constraints aren't met.
type ListsReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ListsReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ListsReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ListsReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ListsReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ListsReplyValidationError) ErrorName() string { return "ListsReplyValidationError" }

// Error satisfies the builtin error interface
func (e ListsReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sListsReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ListsReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ListsReplyValidationError{}

// Validate checks the field values on SaveRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *SaveRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on SaveRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in SaveRequestMultiError, or
// nil if none found.
func (m *SaveRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *SaveRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ProductName

	// no validation rules for DeploymentruntimeName

	// no validation rules for InsecureSkipCheck

	if all {
		switch v := interface{}(m.GetBody()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SaveRequestValidationError{
					field:  "Body",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SaveRequestValidationError{
					field:  "Body",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetBody()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SaveRequestValidationError{
				field:  "Body",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return SaveRequestMultiError(errors)
	}

	return nil
}

// SaveRequestMultiError is an error wrapping multiple validation errors
// returned by SaveRequest.ValidateAll() if the designated constraints aren't met.
type SaveRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SaveRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SaveRequestMultiError) AllErrors() []error { return m }

// SaveRequestValidationError is the validation error returned by
// SaveRequest.Validate if the designated constraints aren't met.
type SaveRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SaveRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SaveRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SaveRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SaveRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SaveRequestValidationError) ErrorName() string { return "SaveRequestValidationError" }

// Error satisfies the builtin error interface
func (e SaveRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSaveRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SaveRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SaveRequestValidationError{}

// Validate checks the field values on SaveReply with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *SaveReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on SaveReply with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in SaveReplyMultiError, or nil
// if none found.
func (m *SaveReply) ValidateAll() error {
	return m.validate(true)
}

func (m *SaveReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Msg

	if len(errors) > 0 {
		return SaveReplyMultiError(errors)
	}

	return nil
}

// SaveReplyMultiError is an error wrapping multiple validation errors returned
// by SaveReply.ValidateAll() if the designated constraints aren't met.
type SaveReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SaveReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SaveReplyMultiError) AllErrors() []error { return m }

// SaveReplyValidationError is the validation error returned by
// SaveReply.Validate if the designated constraints aren't met.
type SaveReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SaveReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SaveReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SaveReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SaveReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SaveReplyValidationError) ErrorName() string { return "SaveReplyValidationError" }

// Error satisfies the builtin error interface
func (e SaveReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSaveReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SaveReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SaveReplyValidationError{}

// Validate checks the field values on DeleteRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *DeleteRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in DeleteRequestMultiError, or
// nil if none found.
func (m *DeleteRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ProductName

	// no validation rules for DeploymentruntimeName

	// no validation rules for InsecureSkipCheck

	if len(errors) > 0 {
		return DeleteRequestMultiError(errors)
	}

	return nil
}

// DeleteRequestMultiError is an error wrapping multiple validation errors
// returned by DeleteRequest.ValidateAll() if the designated constraints
// aren't met.
type DeleteRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteRequestMultiError) AllErrors() []error { return m }

// DeleteRequestValidationError is the validation error returned by
// DeleteRequest.Validate if the designated constraints aren't met.
type DeleteRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteRequestValidationError) ErrorName() string { return "DeleteRequestValidationError" }

// Error satisfies the builtin error interface
func (e DeleteRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteRequestValidationError{}

// Validate checks the field values on DeleteReply with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *DeleteReply) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeleteReply with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in DeleteReplyMultiError, or
// nil if none found.
func (m *DeleteReply) ValidateAll() error {
	return m.validate(true)
}

func (m *DeleteReply) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Msg

	if len(errors) > 0 {
		return DeleteReplyMultiError(errors)
	}

	return nil
}

// DeleteReplyMultiError is an error wrapping multiple validation errors
// returned by DeleteReply.ValidateAll() if the designated constraints aren't met.
type DeleteReplyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeleteReplyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeleteReplyMultiError) AllErrors() []error { return m }

// DeleteReplyValidationError is the validation error returned by
// DeleteReply.Validate if the designated constraints aren't met.
type DeleteReplyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeleteReplyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeleteReplyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeleteReplyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeleteReplyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeleteReplyValidationError) ErrorName() string { return "DeleteReplyValidationError" }

// Error satisfies the builtin error interface
func (e DeleteReplyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeleteReply.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeleteReplyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeleteReplyValidationError{}

// Validate checks the field values on DeploymentRuntimesDestination with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *DeploymentRuntimesDestination) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on DeploymentRuntimesDestination with
// the rules defined in the proto definition for this message. If any rules
// are violated, the result is a list of violation errors wrapped in
// DeploymentRuntimesDestinationMultiError, or nil if none found.
func (m *DeploymentRuntimesDestination) ValidateAll() error {
	return m.validate(true)
}

func (m *DeploymentRuntimesDestination) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if utf8.RuneCountInString(m.GetEnvironment()) < 1 {
		err := DeploymentRuntimesDestinationValidationError{
			field:  "Environment",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return DeploymentRuntimesDestinationMultiError(errors)
	}

	return nil
}

// DeploymentRuntimesDestinationMultiError is an error wrapping multiple
// validation errors returned by DeploymentRuntimesDestination.ValidateAll()
// if the designated constraints aren't met.
type DeploymentRuntimesDestinationMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m DeploymentRuntimesDestinationMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m DeploymentRuntimesDestinationMultiError) AllErrors() []error { return m }

// DeploymentRuntimesDestinationValidationError is the validation error
// returned by DeploymentRuntimesDestination.Validate if the designated
// constraints aren't met.
type DeploymentRuntimesDestinationValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DeploymentRuntimesDestinationValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DeploymentRuntimesDestinationValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DeploymentRuntimesDestinationValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DeploymentRuntimesDestinationValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DeploymentRuntimesDestinationValidationError) ErrorName() string {
	return "DeploymentRuntimesDestinationValidationError"
}

// Error satisfies the builtin error interface
func (e DeploymentRuntimesDestinationValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDeploymentRuntimesDestination.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DeploymentRuntimesDestinationValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DeploymentRuntimesDestinationValidationError{}

// Validate checks the field values on SaveRequest_Body with the rules defined
// in the proto definition for this message. If any rules are violated, the
// first error encountered is returned, or nil if there are no violations.
func (m *SaveRequest_Body) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on SaveRequest_Body with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// SaveRequest_BodyMultiError, or nil if none found.
func (m *SaveRequest_Body) ValidateAll() error {
	return m.validate(true)
}

func (m *SaveRequest_Body) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetManifestSource() == nil {
		err := SaveRequest_BodyValidationError{
			field:  "ManifestSource",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetManifestSource()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SaveRequest_BodyValidationError{
					field:  "ManifestSource",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SaveRequest_BodyValidationError{
					field:  "ManifestSource",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetManifestSource()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SaveRequest_BodyValidationError{
				field:  "ManifestSource",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetDestination()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SaveRequest_BodyValidationError{
					field:  "Destination",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SaveRequest_BodyValidationError{
					field:  "Destination",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetDestination()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SaveRequest_BodyValidationError{
				field:  "Destination",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return SaveRequest_BodyMultiError(errors)
	}

	return nil
}

// SaveRequest_BodyMultiError is an error wrapping multiple validation errors
// returned by SaveRequest_Body.ValidateAll() if the designated constraints
// aren't met.
type SaveRequest_BodyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SaveRequest_BodyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SaveRequest_BodyMultiError) AllErrors() []error { return m }

// SaveRequest_BodyValidationError is the validation error returned by
// SaveRequest_Body.Validate if the designated constraints aren't met.
type SaveRequest_BodyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SaveRequest_BodyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SaveRequest_BodyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SaveRequest_BodyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SaveRequest_BodyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SaveRequest_BodyValidationError) ErrorName() string { return "SaveRequest_BodyValidationError" }

// Error satisfies the builtin error interface
func (e SaveRequest_BodyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSaveRequest_Body.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SaveRequest_BodyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SaveRequest_BodyValidationError{}
