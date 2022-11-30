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

package types

// Error is a generic error returns by the api
type Error struct {
	// Code is an optional machine readable code used to describe the code
	Code int `json:"code"`
	// Detail is the actual error thrown by the upstream
	Detail string `json:"detail"`
	// Message is a human readable message related to the error
	Message string `json:"message"`
	// URI is the uri of the request
	URI string `json:"uri"`
	// Verb was the http request verb used
	Verb string `json:"verb"`
}

// Error returns the error message
func (e Error) Error() string {
	return e.Message
}

// Is reports whether the error message matches the target error message
func (e Error) Is(target error) bool {
	return e.Message == target.Error()
}

// WithVerb sets the request verb
func (e *Error) WithVerb(v string) *Error {
	e.Verb = v

	return e
}

// WithURI sets the request uri
func (e *Error) WithURI(v string) *Error {
	e.URI = v

	return e
}

// WithDetail addes a detailed message
func (e *Error) WithDetail(v string) *Error {
	e.Detail = v

	return e
}

// WithCode adds a code to the api error
func (e *Error) WithCode(v int) *Error {
	e.Code = v

	return e
}
