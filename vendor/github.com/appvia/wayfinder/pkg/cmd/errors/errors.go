/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package errors

import (
	kerrors "errors"
	"fmt"

	reflectutils "github.com/appvia/wayfinder/pkg/utils/reflect"
)

var (
	// ErrAuthentication indicates we need to authenticate
	ErrAuthentication = kerrors.New("authorization required, ensure you have logged in")
	// ErrMissingResource indicates the resource name is missing
	ErrMissingResource = kerrors.New("resource is missing")
	// ErrMissingResourceName indicates the resource name is missing
	ErrMissingResourceName = kerrors.New("name is missing")
	// ErrMissingResourceKind indicates the resource is missing the api kind
	ErrMissingResourceKind = kerrors.New("resource missing api kind")
	// ErrMissingResourceVersion indicates the resource is missing the api version
	ErrMissingResourceVersion = kerrors.New("resource is missing api version")
	// ErrWorkspaceMissing indicates the resource requires a workspace selector
	ErrWorkspaceMissing = kerrors.New("resource is workspace scoped and requires a workspace name")
	// ErrOperationNotPermitted indicates the operation is not permitted
	ErrOperationNotPermitted = kerrors.New("operation not permitted on the resource")
	// ErrOperationNotSupported indicates the operation was not supported
	ErrOperationNotSupported = kerrors.New("operation not supported")
	// ErrMissingProfile indicate the profile does not exist
	ErrMissingProfile = kerrors.New("profile does not exist")
	// ErrNoFiles indicates no resources have been defined
	ErrNoFiles = kerrors.New("no resource file defined")
	// ErrUnknownResource indicates an unknown resource
	ErrUnknownResource = kerrors.New("unknown resource type")
	// ErrUnknownAccountType indicates an unknown account
	ErrUnknownAccountType = kerrors.New("unknown account type")
)

// ErrCustom is a base wrapper for a custom error
type ErrCustom struct {
	message string
}

// ErrResourceNotFound indicates the resources was not found
type ErrResourceNotFound struct {
	resource, kind string
}

// ErrConflict indicates a conflict error
type ErrConflict struct {
	ErrCustom
}

// ErrConfigInvalid indicates a problem with the client configuration
type ErrConfigInvalid struct {
	ErrCustom
}

// ErrProfileInvalid indicates an issue with the profile
type ErrProfileInvalid struct {
	ErrCustom
	profile string
}

// ErrInvalidParameter indicates an invalid param
type ErrInvalidParameter struct {
	field, value, message string
}

// ErrTimeout indicate a operational timeout
type ErrTimeout struct {
	message string
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("conflict: %s", e.message)
}

func (e *ErrResourceNotFound) Error() string {
	return fmt.Sprintf("%s: %q not found", e.kind, e.resource)
}

func (e *ErrInvalidParameter) Error() string {
	if e.message == "" {
		return fmt.Sprintf("invalid parameter: %q, value: %q", e.field, e.value)
	}

	return fmt.Sprintf("%s, field: %q, value: %q", e.message, e.field, e.value)
}

// Profile returns the profile name is was concerning
func (e *ErrProfileInvalid) Profile() string {
	return e.profile
}

func (e *ErrProfileInvalid) Error() string {
	return fmt.Sprintf("%q %s", e.profile, e.message)
}

// IsError checks the error type is eqaul
func IsError(err error, t interface{}) bool {
	return reflectutils.IsEqualType(err, t)
}

// NewConfigError returns a missing profile error
func NewConfigError(message string) error {
	return &ErrConfigInvalid{ErrCustom: ErrCustom{"invalid config: " + message}}
}

// NewConfigErrorf is the same as above but with formatting args
func NewConfigErrorf(message string, args ...interface{}) error {
	return &ErrConfigInvalid{ErrCustom: ErrCustom{"invalid config: " + fmt.Sprintf(message, args...)}}
}

// NewProfileInvalidError returns a profile invalid
func NewProfileInvalidError(message, profile string) error {
	return &ErrProfileInvalid{ErrCustom: ErrCustom{message}, profile: profile}
}

func (e *ErrTimeout) Error() string {
	return fmt.Sprintf("timed out: %s", e.message)
}

func (e *ErrCustom) Error() string {
	return e.message
}

// NewTimeoutError returns a timeout
func NewTimeoutError(message string) error {
	return &ErrTimeout{message: message}
}

// NewResourceNotFound returns a error type
func NewResourceNotFound(resource string) error {
	return &ErrResourceNotFound{resource: resource, kind: "resource"}
}

// NewResourceNotFoundWithKind returns a error type
func NewResourceNotFoundWithKind(resource, kind string) error {
	return &ErrResourceNotFound{resource: resource, kind: kind}
}

// NewConflictError returns a conflict error
func NewConflictError(message string, args ...interface{}) error {
	return &ErrConflict{ErrCustom: ErrCustom{fmt.Sprintf(message, args...)}}
}

// NewInvalidParamError returns a error type
func NewInvalidParamError(field, value string) error {
	return &ErrInvalidParameter{field: field, value: value}
}

// NewInvalidParamWithMessageError returns a error type
func NewInvalidParamWithMessageError(field, value, message string) error {
	return &ErrInvalidParameter{field: field, value: value, message: message}
}
