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

package client

import (
	"net"
	"net/http"
	"strings"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/appvia/wayfinder/pkg/apiserver/types"
)

var (
	// DefaultHTTPClient is the default http client
	DefaultHTTPClient = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
)

// IsNotFound check if error is an 404 error
func IsNotFound(err error) bool {
	return isExpectedError(err, http.StatusNotFound)
}

// IsNotAuthorized checks if the error as a 401
func IsNotAuthorized(err error) bool {
	return isExpectedError(err, http.StatusUnauthorized)
}

// IsNotImplemented check if the error is a 501
func IsNotImplemented(err error) bool {
	return isExpectedError(err, http.StatusNotImplemented)
}

// IsNotAllowed checks if the response was a 403 forbidden
func IsNotAllowed(err error) bool {
	return isExpectedError(err, http.StatusForbidden)
}

// IsServiceUnavailable checks for s 503
func IsServiceUnavailable(err error) bool {
	return isExpectedError(err, http.StatusServiceUnavailable)
}

// IsMethodNotAllowed checks if the response was a 405 forbidden
func IsMethodNotAllowed(err error) bool {
	return isExpectedError(err, http.StatusMethodNotAllowed)
}

// IsBadRequest checks if the response was a 400
func IsBadRequest(err error) bool {
	return isExpectedError(err, http.StatusBadRequest)
}

// isExpectError checks if the error an apiError and compares the code
func isExpectedError(err error, code int) bool {
	e, ok := (err).(*types.Error)
	if !ok {
		return false
	}

	return e.Code == code
}

func IsAlreadyExists(err error) bool {
	return kerrors.IsAlreadyExists(err) ||
		strings.Contains(err.Error(), "already exists") // TODO: Fix error types we return
}

func IsObjectModified(err error) bool {
	return err != nil && err.Error() == ObjectModifiedError
}
