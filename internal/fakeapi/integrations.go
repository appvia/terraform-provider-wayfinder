// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
)

// Feature is a catalogue definition identified by its bare name.
// BlockDisable stands in for the dependents checked by the Integration handler.
type Feature struct {
	Name           string
	AlwaysEnabled  bool
	Custom         bool
	RequiredParams []string
	BoolParams     []string
	BlockDisable   bool
}

// RegisterFeature adds or replaces a catalogue definition. Register the Integration kind first.
func (s *Server) RegisterFeature(f Feature) {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.lookupKind(Ref{Group: "integrations", Version: "v3", APIPath: "integrations"})
	if !ok {
		s.t.Fatalf("fakeapi: RegisterFeature requires the tenant Integration kind to be registered")
	}

	if len(s.features) == 0 {
		validate := k.Validate
		k.Validate = func(op Operation, obj map[string]any) []FieldError {
			errs := s.validateIntegration(op, obj)
			if validate != nil {
				errs = append(errs, validate(op, obj)...)
			}

			return errs
		}
	}

	s.features[f.Name] = f
}

// validateIntegration mirrors server/integrations/handlers/internal/integration/validate.go:
// CheckCreate reserves feature names/types; validateIntegration checks enablement and params.
// ASSUMPTION: BlockDisable models any dependent with one generic refusal rather than a list.
func (s *Server) validateIntegration(op Operation, obj map[string]any) []FieldError {
	name := str(metadata(obj), "name")
	spec := child(obj, "spec", false)

	var errs []FieldError

	switch op {
	case OpCreate:
		if strings.HasPrefix(name, "wayfinder.") {
			errs = append(errs, FieldError{Field: "metadata.name", ErrCode: "invalidValue", Message: "Names beginning with \"wayfinder.\" are reserved for Wayfinder-supplied integrations. Enable one with the enable action rather than creating it directly"})
		}

		if f, ok := s.features[name]; ok && !f.Custom {
			errs = append(errs, FieldError{Field: "metadata.name", ErrCode: "invalidValue", Message: fmt.Sprintf("Name %q is reserved for the Wayfinder %q integration. Choose another name", name, name)})
		}

		if str(spec, "type") == "feature" {
			errs = append(errs, FieldError{Field: "spec.type", ErrCode: "invalidValue", Message: "You cannot create a feature integration. Features are turned on with the enable action, not created. Create a custom integration instead"})
		}
	case OpUpdate:
		f, ok := s.features[strings.TrimPrefix(name, "wayfinder.")]
		if !strings.HasPrefix(name, "wayfinder.") || !ok {
			return nil
		}

		enabled, _ := spec["enabled"].(bool)
		if !enabled && f.AlwaysEnabled {
			errs = append(errs, FieldError{Field: "spec.enabled", ErrCode: "invalidValue", Message: fmt.Sprintf("Integration %q is always enabled and cannot be disabled", f.Name)})
		}

		if !enabled && f.BlockDisable {
			errs = append(errs, FieldError{Field: "spec.enabled", ErrCode: "invalidValue", Message: fmt.Sprintf("Integration %q cannot be disabled while resources depend on it", f.Name)})
		}

		if enabled {
			params := child(spec, "params", false)
			for _, key := range f.RequiredParams {
				if params[key] == nil {
					errs = append(errs, FieldError{Field: "spec.params." + key, ErrCode: "required", Message: key + " is required"})
				}
			}
		}
	}

	return errs
}

// handleEnable mirrors pkg/apiserver/handlers/integrations.go:87-96 and
// server/org/services/integrations/enable.go: Enable creates or re-enables a feature,
// merging params without removing omitted keys. Bool coercion mirrors values.CoerceParams.
func (s *Server) handleEnable(w http.ResponseWriter, r *http.Request, segs []string, body []byte) {
	bare := segs[2]
	f, ok := s.features[bare]

	if !ok {
		writeError(w, r, http.StatusNotFound, fmt.Sprintf("integration %q does not exist", bare))
		return
	}

	if f.Custom {
		writeValidation(w, "cannot enable integration", []FieldError{{Field: "name", ErrCode: "notAllowed", Message: fmt.Sprintf("%q is a custom integration and is installed, not enabled; create it from its template with the create action", bare)}})
		return
	}

	var req struct {
		Params map[string]json.RawMessage `json:"params"`
	}

	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	for _, key := range f.BoolParams {
		var value string
		if json.Unmarshal(req.Params[key], &value) == nil && (value == "true" || value == "false") {
			req.Params[key] = json.RawMessage(value)
		}
	}

	ref := Ref{Tenant: segs[0], Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder." + bare}
	k, _ := s.lookupKind(ref)
	obj := map[string]any{
		"apiVersion": "integrations/v3",
		"kind":       "Integration",
		"metadata":   map[string]any{"name": ref.Name},
		"spec":       map[string]any{"type": "feature", "enabled": true},
	}
	e, exists := s.objects[ref]

	if exists {
		obj = deepCopy(e.obj)
	}

	spec := child(obj, "spec", true)
	spec["enabled"] = true

	if len(req.Params) > 0 {
		params := make(map[string]any, len(req.Params))
		for key, value := range req.Params {
			params[key] = value
		}

		maps.Copy(child(spec, "params", true), params)
	}

	// Normalise RawMessages before validation, including JSON null required params.
	obj = deepCopy(obj)
	if !runValidate(w, k, OpUpdate, obj) {
		return
	}

	if exists {
		s.updateEntry(k, e, obj)
	} else {
		e = s.newEntry(k, obj)
		s.objects[ref] = e
	}

	s.respond(w, e)
}

// handleEffective mirrors EffectiveIntegration in
// server/integrations/apis/integrations/v3/integration_summary_types.go:117-133.
// ASSUMPTION: unconfigured features project disabled; catalogue defaults are not modelled.
func (s *Server) handleEffective(w http.ResponseWriter, r *http.Request, segs []string) {
	bare := segs[2]
	f, ok := s.features[bare]

	if !ok {
		writeError(w, r, http.StatusNotFound, fmt.Sprintf("integration %q does not exist", bare))
		return
	}

	ref := Ref{Tenant: segs[0], Group: "integrations", Version: "v3", APIPath: "integrations", Name: "wayfinder." + bare}
	obj := map[string]any{
		"apiVersion": "integrations/v3",
		"kind":       "Integration",
		"metadata":   map[string]any{"name": ref.Name},
		"spec":       map[string]any{"type": "feature", "enabled": false},
	}
	e, exists := s.objects[ref]

	if exists {
		obj = deepCopy(e.obj)
		sanitize(e.kind, obj)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"integration": obj, "exists": exists, "available": true, "alwaysEnabled": f.AlwaysEnabled,
	})
}
