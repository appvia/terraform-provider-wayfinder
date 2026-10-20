// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// VarScope identifies where a per-key variable or secret lives (R16).
type VarScope struct {
	Tenant, Workspace, Environment, Instance string
}

// varEntry mirrors secretsv3.SecretEntry / VariableEntry
// (server/org/apis/secrets/v3/varset_types.go:75-100).
type varEntry struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value,omitempty"`
}

// varSet stands in for the scope's VarSet; it exists once anything was set there.
type varSet struct {
	Secrets   []varEntry `json:"secrets,omitempty"`
	Variables []varEntry `json:"variables,omitempty"`
}

// parseVarsPath parses the vars scopes served under a tenant:
//   - /api/v3/general/tenants/{t}[/workspaces/{ws}[/envs/{env}]]/vars[/{secrets|variables}/{key}]
//   - /api/v3/general/tenants/{t}/workspaces/{ws}/envs/{env}/instancevars/{instance}[/{secrets|variables}/{key}]
//
// The instance scope has no /vars segment: its base is the instancevars/{instance} path itself.
func parseVarsPath(r *http.Request) (scope VarScope, kind, key string, ok bool) {
	segs, ok := splitEscaped(strings.TrimPrefix(r.URL.EscapedPath(), tenantsPrefix))
	if !ok || len(segs) < 2 {
		return VarScope{}, "", "", false
	}
	scope.Tenant, segs = segs[0], segs[1:]
	if len(segs) >= 2 && segs[0] == "workspaces" {
		scope.Workspace, segs = segs[1], segs[2:]
		if len(segs) >= 2 && segs[0] == "envs" {
			scope.Environment, segs = segs[1], segs[2:]
		}
	}
	instance := scope.Environment != "" && len(segs) >= 2 && segs[0] == "instancevars"
	if instance {
		scope.Instance, segs = segs[1], segs[2:]
	}
	switch {
	case instance && len(segs) == 0:
		return scope, "", "", true
	case instance && len(segs) == 2 && (segs[0] == "secrets" || segs[0] == "variables"):
		return scope, segs[0], segs[1], true
	case instance:
		return VarScope{}, "", "", false
	case len(segs) == 1 && segs[0] == "vars":
		return scope, "", "", true
	case len(segs) == 3 && segs[0] == "vars" && (segs[1] == "secrets" || segs[1] == "variables"):
		return scope, segs[1], segs[2], true
	default:
		return VarScope{}, "", "", false
	}
}

func (v *varSet) list(kind string) *[]varEntry {
	if kind == "secrets" {
		return &v.Secrets
	}
	return &v.Variables
}

// handleVars mirrors pkg/apiserver/handlers/vars.go:274-398 and the VarManager
// (server/org/services/vars/varmanager.go:150-262):
//   - GET .../vars: the VarSetSpec with secret values removed; `{}` when nothing was set;
//   - GET a key: 404 when the set or key is missing; variables return types.VariableItem
//     {key,value}; secrets return {key} with value null (never disclosed);
//   - PUT a key: body types.SetValueRequest {value}, response types.SetValueResponse {key};
//   - DELETE a key: 404 when the scope has no set; a missing key is a no-op; 204 No Content.
func (s *Server) handleVars(w http.ResponseWriter, r *http.Request, body []byte) {
	scope, kind, key, ok := parseVarsPath(r)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	set := s.vars[scope]
	switch {
	case kind == "" && r.Method == http.MethodGet:
		masked := varSet{}
		if set != nil {
			for _, e := range set.Secrets {
				masked.Secrets = append(masked.Secrets, varEntry{Name: e.Name})
			}
			masked.Variables = set.Variables
		}
		writeJSON(w, http.StatusOK, masked)
	case kind != "" && r.Method == http.MethodGet:
		e := lookupVar(set, kind, key)
		if e == nil {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		item := map[string]any{"key": key, "value": nil}
		if kind == "variables" {
			item["value"] = e.Value
		}
		writeJSON(w, http.StatusOK, item)
	case kind != "" && r.Method == http.MethodPut:
		var req struct {
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid JSON body")
			return
		}
		s.setVar(scope, kind, key, req.Value)
		writeJSON(w, http.StatusOK, map[string]any{"key": key})
	case kind != "" && r.Method == http.MethodDelete:
		if set == nil {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		l := set.list(kind)
		*l = slices.DeleteFunc(*l, func(e varEntry) bool { return e.Name == key })
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func lookupVar(set *varSet, kind, key string) *varEntry {
	if set == nil {
		return nil
	}
	l := *set.list(kind)
	if i := slices.IndexFunc(l, func(e varEntry) bool { return e.Name == key }); i >= 0 {
		return &l[i]
	}
	return nil
}

func (s *Server) setVar(scope VarScope, kind, key string, value json.RawMessage) {
	set := s.vars[scope]
	if set == nil {
		set = &varSet{}
		s.vars[scope] = set
	}
	if e := lookupVar(set, kind, key); e != nil {
		e.Value = slices.Clone(value)
		return
	}
	l := set.list(kind)
	*l = append(*l, varEntry{Name: key, Value: slices.Clone(value)})
}

func (s *Server) getVar(scope VarScope, kind, key string) (json.RawMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := lookupVar(s.vars[scope], kind, key); e != nil {
		return slices.Clone(e.Value), true
	}
	return nil, false
}

// Secret returns the plaintext JSON value recorded for a secret (for test assertions only).
func (s *Server) Secret(scope VarScope, key string) (json.RawMessage, bool) {
	return s.getVar(scope, "secrets", key)
}

// Variable returns the JSON value of a variable.
func (s *Server) Variable(scope VarScope, key string) (json.RawMessage, bool) {
	return s.getVar(scope, "variables", key)
}

// SetSecret seeds a secret directly, bypassing the API.
func (s *Server) SetSecret(scope VarScope, key string, value json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setVar(scope, "secrets", key, value)
}

// SetVariable seeds a variable directly, bypassing the API.
func (s *Server) SetVariable(scope VarScope, key string, value json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setVar(scope, "variables", key, value)
}
