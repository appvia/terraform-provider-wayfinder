// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Scope is an object scope as named in the manifest contract.
type Scope string

// Object scopes (pkg/apis/core/v3/object_scope_types.go:21-168).
const (
	ScopeTenant      Scope = "tenant"
	ScopeWorkspace   Scope = "workspace"
	ScopeEnvironment Scope = "environment"
)

// Operation identifies the write a Validate hook is called for.
type Operation string

// Write operations passed to KindSpec.Validate.
const (
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
)

// FieldError is one entry of a validation.Error body (ui/api-swagger.json validation.FieldError).
type FieldError struct {
	Field   string `json:"field"`
	ErrCode string `json:"errCode"`
	Message string `json:"message"`
}

// StatusStep is one observed status in a status script.
type StatusStep struct {
	Status  string
	Message string
	Detail  string
}

// KindSpec registers a kind with the fake. Paths are dotted JSON paths from the object root
// (e.g. "spec.bearerToken.token"); when a path segment is an array, the rest of the path applies
// to every element (fields.yaml convention).
type KindSpec struct {
	Kind      string
	Group     string
	Version   string
	APIPath   string
	Scopes    []Scope
	Versioned bool
	HasStatus bool
	// SensitiveStrings are +encrypted string leaves; SensitiveMaps are +encrypted
	// map[string]string leaves (R18).
	SensitiveStrings []string
	SensitiveMaps    []string
	// Defaults maps a dotted path to the value the server fills in on create when absent.
	Defaults map[string]any
	// FinalizerGETs > 0 enables finalizer mode: DELETE sets metadata.deletionTimestamp and the
	// object stays readable for that many GETs before it is gone.
	FinalizerGETs int
	// StatusScript is the default status progression for objects of this kind. Empty means a
	// single reconciled Success step.
	StatusScript []StatusStep
	// Validate, when set, runs on the submitted body of every create/update; any returned field
	// errors produce a 400 validation.Error and the write is rejected.
	Validate func(op Operation, obj map[string]any) []FieldError
}

// Ref identifies one stored object. ObjectVersion is set only for versioned kinds.
type Ref struct {
	Tenant, Workspace, Environment string
	Group, Version, APIPath        string
	Name, ObjectVersion            string
}

type kindKey struct{ group, version, apiPath string }

func (k *KindSpec) key() kindKey { return kindKey{k.Group, k.Version, k.APIPath} }

func (r Ref) kindKey() kindKey { return kindKey{r.Group, r.Version, r.APIPath} }

func (r Ref) scope() Scope {
	switch {
	case r.Environment != "":
		return ScopeEnvironment
	case r.Workspace != "":
		return ScopeWorkspace
	default:
		return ScopeTenant
	}
}

// RegisterKind adds or replaces a kind definition.
func (s *Server) RegisterKind(k KindSpec) {
	s.t.Helper()

	if k.Group == "" || k.Version == "" || k.APIPath == "" || len(k.Scopes) == 0 {
		s.t.Fatalf("fakeapi: kind %q needs group, version, apiPath and scopes", k.Kind)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.kinds[k.key()] = &k
}

// resourcePath is a parsed resource URL (R8, pkg/wfclient/apiurlmanager.go:147-216):
// /api/v3/resources/tenants/{t}[/workspaces/{ws}[/envs/{env}]]/{group}/{version}/{apiPath}[/{name}[/versions/{v}][/{action}]].
type resourcePath struct {
	ref           Ref    // Name empty for collection routes
	hasVersionURL bool   // the /versions/{v} suffix was present
	action        string // trailing /{action} segment of a subresource route
}

func parseResourcePath(r *http.Request) (resourcePath, bool) {
	segs, ok := splitEscaped(strings.TrimPrefix(r.URL.EscapedPath(), resourcesPrefix))
	if !ok || len(segs) < 2 || segs[0] != "tenants" {
		return resourcePath{}, false
	}

	p := resourcePath{ref: Ref{Tenant: segs[1]}}

	rest := segs[2:]
	if len(rest) >= 2 && rest[0] == "workspaces" {
		p.ref.Workspace, rest = rest[1], rest[2:]
		if len(rest) >= 2 && rest[0] == "envs" {
			p.ref.Environment, rest = rest[1], rest[2:]
		}
	}

	if len(rest) < 3 {
		return resourcePath{}, false
	}

	p.ref.Group, p.ref.Version, p.ref.APIPath = rest[0], rest[1], rest[2]
	switch rest = rest[3:]; {
	case len(rest) == 0:
	case len(rest) == 1:
		p.ref.Name = rest[0]
	case len(rest) == 2:
		p.ref.Name, p.action = rest[0], rest[1]
	case len(rest) == 3 && rest[1] == "versions":
		p.ref.Name, p.ref.ObjectVersion, p.hasVersionURL = rest[0], rest[2], true
	case len(rest) == 4 && rest[1] == "versions":
		p.ref.Name, p.ref.ObjectVersion, p.hasVersionURL, p.action = rest[0], rest[2], true, rest[3]
	default:
		return resourcePath{}, false
	}

	return p, true
}

func splitEscaped(p string) ([]string, bool) {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	for i, part := range parts {
		dec, err := url.PathUnescape(part)
		if err != nil || dec == "" {
			return nil, false
		}

		parts[i] = dec
	}

	return parts, true
}

// lookupKind returns the kind for a ref if it is registered and supports the ref's scope.
func (s *Server) lookupKind(ref Ref) (*KindSpec, bool) {
	k, ok := s.kinds[ref.kindKey()]
	if !ok || !slices.Contains(k.Scopes, ref.scope()) {
		return nil, false
	}

	return k, true
}

// strictSemver mirrors the check in pkg/apihandlers/checks.go:87-106 (X.Y.Z with optional
// pre-release/build, no "v" prefix).
var strictSemver = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// writeValidation writes a 400 validation.Error body (ui/api-swagger.json validation.Error;
// decoded by pkg/wfclient/apiclient.go:824-840).
func writeValidation(w http.ResponseWriter, message string, errs []FieldError) {
	writeJSON(w, http.StatusBadRequest, map[string]any{"message": message, "fieldErrors": errs})
}
