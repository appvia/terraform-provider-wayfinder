// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Wire headers distinguishing 409 conflicts (pkg/common/errors.go, R9, R10).
const (
	headerObjectModified      = "x-wayfinder-objectmodified"
	headerObjectAlreadyExists = "x-wayfinder-objectalreadyexists"
)

// Sentinels matched with errors.Is against errors returned by the client.
var (
	// ErrNotFound: 404 (pkg/apiserver/apihelpers/errors.go:122-123).
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists: 409 with x-wayfinder-objectalreadyexists (errors.go:135-137).
	ErrAlreadyExists = errors.New("already exists")
	// ErrModified: 409 with x-wayfinder-objectmodified: true (errors.go:82-92).
	ErrModified = errors.New("object modified")
)

// maxMessageBytes caps the raw body used as a message when the server did not send JSON.
const maxMessageBytes = 1024

// APIError is a non-2xx response other than a 400 validation failure. Message is the server's
// message verbatim; Detail is the optional upstream detail.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
	Detail     string
	sentinel   error
}

func (e *APIError) Error() string {
	msg := e.Message
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	return fmt.Sprintf("wayfinder API %s %s: %d %s: %s", e.Method, e.Path, e.StatusCode, http.StatusText(e.StatusCode), msg)
}

// Is reports whether target is the sentinel this response maps to.
func (e *APIError) Is(target error) bool { return e.sentinel != nil && target == e.sentinel }

// FieldError is one entry of a validation.Error body (pkg/utils/validation/error.go FieldError).
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"errCode"`
	Message string `json:"message"`
}

// ValidationError is a 400 response carrying a validation.Error body.
type ValidationError struct {
	Method  string
	Path    string
	Message string       `json:"message"`
	Fields  []FieldError `json:"fieldErrors"`
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "wayfinder API %s %s: validation failed", e.Method, e.Path)
	if e.Message != "" {
		b.WriteString(": " + e.Message)
	}
	for _, f := range e.Fields {
		fmt.Fprintf(&b, "; %s: %s", f.Field, f.Message)
	}
	return b.String()
}

// errorBody is the generic types.Error body (pkg/apiserver/types/errors.go).
type errorBody struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

// decodeError maps a non-2xx response to a typed error.
func decodeError(method, path string, resp *http.Response, body []byte) error {
	if resp.StatusCode == http.StatusBadRequest {
		v := &ValidationError{}
		if json.Unmarshal(body, v) == nil && (v.Message != "" || len(v.Fields) > 0) {
			v.Method, v.Path = method, path
			return v
		}
	}
	e := &APIError{Method: method, Path: path, StatusCode: resp.StatusCode}
	var eb errorBody
	if json.Unmarshal(body, &eb) == nil && eb.Message != "" {
		e.Message, e.Detail = eb.Message, eb.Detail
	} else {
		raw := strings.TrimSpace(string(body))
		if len(raw) > maxMessageBytes {
			raw = raw[:maxMessageBytes] + "..."
		}
		e.Message = raw
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		e.sentinel = ErrNotFound
	case resp.StatusCode == http.StatusConflict && resp.Header.Get(headerObjectModified) == "true":
		e.sentinel = ErrModified
	case resp.StatusCode == http.StatusConflict && resp.Header.Get(headerObjectAlreadyExists) != "":
		e.sentinel = ErrAlreadyExists
	}
	return e
}
