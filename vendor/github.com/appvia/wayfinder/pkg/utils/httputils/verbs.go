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

package httputils

import (
	"errors"
	"net/http"
)

var ErrUnsupportedVerb = errors.New("supported verb")

// AllMethods contains all http methods
var AllMethods = []string{
	http.MethodDelete,
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
}

// GetVerb returns the default verb for a request
func GetVerb(method string) (string, error) {
	switch method {
	case http.MethodGet:
		method = "get"
	case http.MethodDelete:
		method = "delete"
	case http.MethodPost:
		method = "create"
	case http.MethodPut:
		method = "update"
	case http.MethodPatch:
		method = "patch"
	default:
		return "", ErrUnsupportedVerb
	}

	return method, nil
}
