// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"net/http"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
)

// subKey identifies one canned subresource response: an object and the action segment that
// follows its item path (client.Ref.ItemPath + "/" + action).
type subKey struct {
	ref    client.Ref
	action string
}

type subresource struct {
	status int
	body   any
}

// SetSubresource makes GET <ref.ItemPath>/<action> answer status with body (JSON encoded).
// client.Ref carries no tenant, so the response is served for every tenant. The object need not
// exist. Unset subresources answer 404, and the state belongs to this Server only.
func (s *Server) SetSubresource(ref client.Ref, action string, status int, body any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subresources == nil {
		s.subresources = map[subKey]subresource{}
	}

	s.subresources[subKey{ref: ref, action: action}] = subresource{status: status, body: body}
}

// getSubresource serves item subresource routes. Only GET is supported; anything else, an
// unknown kind, a versioned kind without the /versions/{v} suffix (client.Ref.ItemPath rejects
// it) or an unset subresource is a 404, as for any unrouted path.
func (s *Server) getSubresource(w http.ResponseWriter, r *http.Request, p resourcePath) {
	k, known := s.lookupKind(p.ref)
	if !known || r.Method != http.MethodGet || (k.Versioned && !p.hasVersionURL) {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}

	key := subKey{action: p.action, ref: client.Ref{
		Group: p.ref.Group, Version: p.ref.Version, APIPath: p.ref.APIPath,
		Workspace: p.ref.Workspace, Environment: p.ref.Environment,
		Name: p.ref.Name, ObjectVersion: p.ref.ObjectVersion, Versioned: k.Versioned,
	}}

	sub, set := s.subresources[key]
	if !set {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, sub.status, sub.body)
}
