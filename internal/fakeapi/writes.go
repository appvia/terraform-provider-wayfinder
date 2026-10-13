package fakeapi

import (
	"net/http"
	"strconv"
	"time"
)

// update: PUT on the item. Missing -> 404 (pkg/apihandlers/wrappers.go:149-158,
// wfclientstore.go:997-998; the fake never honours apply=true). A non-zero generation that
// differs from the stored one -> 409 + objectmodified header + OBJECT_MODIFIED body
// (wfclientstore.go:1016-1022, errors.go:82-92); generation 0 skips the check. Blank secrets are
// backfilled from the stored object, then generation is incremented.
// ASSUMPTION: the server keeps uid, creationTimestamp, deletionTimestamp and status from the
// stored object and ignores client-sent values for them.
func (s *Server) update(w http.ResponseWriter, r *http.Request, k *KindSpec, ref Ref, body []byte) {
	obj, ok := decodeObject(w, r, k, &ref, body)
	if !ok || !runValidate(w, k, OpUpdate, obj) {
		return
	}
	e, exists := s.objects[ref]
	if !exists {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	stored := generation(e.obj)
	if sent := generation(obj); sent != 0 && sent != stored {
		w.Header().Set(HeaderObjectModified, "true")
		writeJSON(w, http.StatusConflict, map[string]any{
			"message": "OBJECT_MODIFIED",
			"detail":  "generation mismatch: provided " + strconv.FormatInt(sent, 10) + ", current stored generation " + strconv.FormatInt(stored, 10),
		})
		return
	}
	mergeSecrets(k, obj, e.obj)
	md, prev := metadata(obj), metadata(e.obj)
	for _, f := range []string{"uid", "creationTimestamp", "deletionTimestamp"} {
		if v, ok := prev[f]; ok {
			md[f] = v
		} else {
			delete(md, f)
		}
	}
	md["generation"] = stored + 1
	prevStatus := child(e.obj, "status", false)
	e.obj = obj
	s.startOperation(e, prevStatus)
	e.obj = deepCopy(e.obj)
	s.respond(w, e)
}

// remove: DELETE on the item. Missing -> 404. Without finalizer mode the object is gone at once;
// with it, deletionTimestamp is set and the object stays readable (wfclientstore.go:1280-1301).
// The response is the object (pkg/apihandlers/apiobjecthandler_delete.go:135).
func (s *Server) remove(w http.ResponseWriter, r *http.Request, ref Ref) {
	e, ok := s.objects[ref]
	if !ok {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if e.kind.FinalizerGETs > 0 {
		if !e.deleting {
			e.deleting, e.getsLeft = true, e.kind.FinalizerGETs
			metadata(e.obj)["deletionTimestamp"] = now().Format(time.RFC3339)
		}
	} else {
		delete(s.objects, ref)
	}
	s.respond(w, e)
}

// Seed stores obj directly, bypassing the API (no validation, no request log). Server-owned
// metadata (generation 1, uid, creationTimestamp) is filled in only when absent; secrets are
// stored as given. Scope fields in metadata are set from ref.
func (s *Server) Seed(ref Ref, obj map[string]any) {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.lookupKind(ref)
	if !ok {
		s.t.Fatalf("fakeapi: Seed: kind %s/%s/%s not registered for scope %s", ref.Group, ref.Version, ref.APIPath, ref.scope())
	}
	o := deepCopy(obj)
	md := metadata(o)
	md["name"] = ref.Name
	for f, v := range map[string]string{"workspace": ref.Workspace, "environment": ref.Environment, "version": ref.ObjectVersion} {
		if v != "" {
			md[f] = v
		}
	}
	if generation(o) == 0 {
		md["generation"] = 1
	}
	for f, v := range map[string]string{"uid": s.nextID("uid"), "creationTimestamp": now().Format(time.RFC3339)} {
		if str(md, f) == "" {
			md[f] = v
		}
	}
	e := &entry{kind: k, obj: o}
	s.startOperation(e, child(o, "status", false))
	e.obj = deepCopy(e.obj)
	s.objects[ref] = e
}

// Mutate simulates an out-of-band edit (e.g. in the UI): fn edits the stored object (secrets
// included), then generation is incremented and a new operation starts.
func (s *Server) Mutate(ref Ref, fn func(obj map[string]any)) {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.mustEntry(ref)
	gen := generation(e.obj)
	o := deepCopy(e.obj)
	fn(o)
	metadata(o)["generation"] = gen + 1
	e.obj = o
	s.startOperation(e, child(o, "status", false))
	e.obj = deepCopy(e.obj)
}

// Object returns a copy of the stored object, secrets included.
func (s *Server) Object(ref Ref) (map[string]any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.objects[ref]
	if !ok {
		return nil, false
	}
	return deepCopy(e.obj), true
}

func (s *Server) mustEntry(ref Ref) *entry {
	s.t.Helper()
	e, ok := s.objects[ref]
	if !ok {
		s.t.Fatalf("fakeapi: no object %+v", ref)
	}
	return e
}
