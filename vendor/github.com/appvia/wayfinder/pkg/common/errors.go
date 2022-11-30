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

package common

import (
	"errors"
)

var (
	// ErrUnsupported indicates the operation or feature is not supported
	ErrUnsupported = errors.New("operation or feature not supported")
	// ErrNotImplemented indicates the operation is not implemented
	ErrNotImplemented = errors.New("operation not implemented")
	// ErrOnRevision indicates the object is older than the revision in source.
	// They need to retrieve the latest copy and retry
	ErrOnRevision = errors.New("object revision is too old")
	// ErrNotFound the object has not been found i.e. doesn't exist
	ErrNotFound = errors.New("requested object does not exist")
	// ErrUnauthorized indicates we don't have permission to the object
	ErrUnauthorized = errors.New("object forbidden, require additional permissions")
	// ErrRequestInvalid indicates an invalid request object
	ErrRequestInvalid = errors.New("invalid request")
	// ErrServerData indicates invalid data
	ErrServerData = errors.New("invalid server data")
	// ErrServerNotAvailable indicates that the remote server is unavailable
	ErrServerNotAvailable = errors.New("a remote server is unavailable")
	// ErrAlreadyExists means the object cannot be created as it already exists
	ErrAlreadyExists = errors.New("object already exists, please use update instead")
)

// NewErrNotAllowed returns an new not allowed error
func NewErrNotAllowed(message string) error {
	return ErrNotAllowed{message: message}
}

// ErrNotAllowed indicates an error of not allowed
type ErrNotAllowed struct {
	message string
}

// Error returns the message
func (e ErrNotAllowed) Error() string {
	if e.message == "" {
		return "object has been denied"
	}

	return e.message
}

type errNotFound struct {
	err error
}

func NewErrNotFound(err error) error {
	return errNotFound{err: err}
}

func (r errNotFound) Error() string {
	return r.err.Error()
}

func IsErrNotFound(err error) bool {
	if err == ErrNotFound || err.Error() == ErrNotFound.Error() {
		return true
	}
	if _, ok := err.(errNotFound); ok {
		return true
	}
	return errors.As(err, &errNotFound{})
}

func IsErrNotAllowed(err error) bool {
	if _, ok := err.(ErrNotAllowed); ok {
		return true
	}
	return errors.As(err, &ErrNotAllowed{})
}
