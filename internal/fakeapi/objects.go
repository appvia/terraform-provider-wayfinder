// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import (
	"net/http"
	"sort"
	"time"
)

func (s *Server) handleResource(w http.ResponseWriter, r *http.Request, body []byte) {
	p, ok := parseResourcePath(r)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}

	k, ok := s.lookupKind(p.ref)
	// Versioned routes exist only for versioned kinds (pkg/apihandlers/apiobjecthandler.go:285-328).
	if !ok || (p.hasVersionURL && !k.Versioned) {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}

	if p.action != "" {
		s.getSubresource(w, r, p)
		return
	}
	// The unversioned item routes of a versioned kind fail with a plain error -> 500
	// (get: pkg/apihandlers/apiobjecthandler.go:503-505; update: apiobjecthandler_update.go:85-87).
	if k.Versioned && p.ref.Name != "" && !p.hasVersionURL {
		if r.Method == http.MethodDelete {
			// ASSUMPTION: delete-all-versions (deleteAllVersionedRouteBuilder) is not faked.
			writeError(w, r, http.StatusNotImplemented, "fakeapi: delete of all versions not implemented")
			return
		}

		writeError(w, r, http.StatusInternalServerError, "internal server error")

		return
	}

	switch {
	case p.ref.Name == "" && r.Method == http.MethodPost:
		s.create(w, r, k, p.ref, body)
	case p.ref.Name == "" && r.Method == http.MethodGet:
		s.list(w, k, p.ref)
	case p.ref.Name != "" && r.Method == http.MethodGet:
		s.get(w, r, p.ref)
	case p.ref.Name != "" && r.Method == http.MethodPut:
		s.update(w, r, k, p.ref, body)
	case p.ref.Name != "" && r.Method == http.MethodDelete:
		s.remove(w, r, p.ref)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// decodeObject parses a write body and checks it against the route: route scope is
// authoritative (pkg/apihandlers/apiobjecthandler_update.go:97-106) and versioned objects need
// a strict semver version matching the URL (apiobjecthandler_update.go:109-119, checks.go:87-106).
// ASSUMPTION: create applies the same workspace/environment checks as update, filling them from
// the URL when the body leaves them empty.
func decodeObject(w http.ResponseWriter, r *http.Request, k *KindSpec, ref *Ref, body []byte) (map[string]any, bool) {
	obj := map[string]any{}
	if err := decodeJSON(body, &obj); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}

	md := metadata(obj)

	for field, want := range map[string]string{"workspace": ref.Workspace, "environment": ref.Environment} {
		if want == "" {
			continue
		}

		if str(md, field) == "" {
			md[field] = want
		}

		if str(md, field) != want {
			writeValidation(w, "Resource validation failed", []FieldError{{Field: "metadata." + field, ErrCode: "invalidValue", Message: field + " must be " + want}})
			return nil, false
		}
	}

	if ref.Name == "" {
		ref.Name = str(md, "name")
		if ref.Name == "" {
			writeValidation(w, "Resource validation failed", []FieldError{{Field: "metadata.name", ErrCode: "required", Message: "name must be defined"}})
			return nil, false
		}
	} else if str(md, "name") != ref.Name {
		// pkg/apihandlers/apiobjecthandler_update.go:93-95: plain error -> 500.
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return nil, false
	}

	if k.Versioned {
		if ref.ObjectVersion != "" && str(md, "version") == "" {
			md["version"] = ref.ObjectVersion
		}

		v := str(md, "version")
		if ref.ObjectVersion != "" && v != ref.ObjectVersion {
			writeValidation(w, "incorrect version set", []FieldError{{Field: "metadata.version", ErrCode: "invalidValue", Message: "must match version provided in URL"}})
			return nil, false
		}

		if !strictSemver.MatchString(v) {
			writeValidation(w, "Resource validation failed", []FieldError{{Field: "metadata.version", ErrCode: "invalidValue", Message: "version must be strict semver X.Y.Z"}})
			return nil, false
		}

		ref.ObjectVersion = v
	}

	return obj, true
}

func runValidate(w http.ResponseWriter, k *KindSpec, op Operation, obj map[string]any) bool {
	if k.Validate == nil {
		return true
	}

	if errs := k.Validate(op, deepCopy(obj)); len(errs) > 0 {
		writeValidation(w, "Resource validation failed", errs)
		return false
	}

	return true
}

// create: POST to the collection. Existing -> 409 + objectalreadyexists header
// (pkg/apiserver/apihelpers/errors.go:135-137, body common.ErrAlreadyExistsData).
func (s *Server) create(w http.ResponseWriter, r *http.Request, k *KindSpec, ref Ref, body []byte) {
	obj, ok := decodeObject(w, r, k, &ref, body)
	if !ok || !runValidate(w, k, OpCreate, obj) {
		return
	}

	if _, exists := s.objects[ref]; exists {
		w.Header().Set(HeaderObjectAlreadyExists, "true")
		writeJSON(w, http.StatusConflict, map[string]any{"message": k.Kind + " " + ref.Name + " already exists"})

		return
	}

	s.objects[ref] = s.newEntry(k, obj)
	s.respond(w, s.objects[ref])
}

// newEntry stamps server-owned metadata on a new object (R6: generation starts at 1).
func (s *Server) newEntry(k *KindSpec, obj map[string]any) *entry {
	md := metadata(obj)
	md["generation"] = 1
	md["uid"] = s.nextID("uid")
	md["creationTimestamp"] = now().Format(time.RFC3339)
	delete(md, "deletionTimestamp")
	child(md, "labels", true)[ManagedByLabel] = "true"

	applyDefaults(k, obj)
	e := &entry{kind: k, obj: obj}
	s.startOperation(e, nil)
	e.obj = deepCopy(e.obj)

	return e
}

func (s *Server) respond(w http.ResponseWriter, e *entry) {
	out := deepCopy(e.obj)
	sanitize(e.kind, out)
	writeJSON(w, http.StatusOK, out)
}

// get: 404 when missing (errors.go:122-123). Each GET advances the status script. Objects being
// deleted with finalizers stay readable (wfclientstore.go:1280-1301) for KindSpec.FinalizerGETs GETs.
func (s *Server) get(w http.ResponseWriter, r *http.Request, ref Ref) {
	e, ok := s.objects[ref]
	if ok && e.deleting {
		if e.getsLeft == 0 {
			delete(s.objects, ref)

			ok = false
		}

		e.getsLeft--
	}

	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}

	if !e.deleting {
		e.advance()
	}

	s.respond(w, e)
}

func (s *Server) list(w http.ResponseWriter, k *KindSpec, scope Ref) {
	refs := make([]Ref, 0)

	for ref, e := range s.objects {
		if e.kind == k && ref.Tenant == scope.Tenant && ref.Workspace == scope.Workspace && ref.Environment == scope.Environment {
			refs = append(refs, ref)
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Name+"@"+refs[i].ObjectVersion < refs[j].Name+"@"+refs[j].ObjectVersion
	})

	items := make([]any, 0, len(refs))
	for _, ref := range refs {
		out := deepCopy(s.objects[ref].obj)
		sanitize(k, out)
		items = append(items, out)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
