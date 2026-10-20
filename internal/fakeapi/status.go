// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi

import "time"

// entry is one stored object plus the fake's bookkeeping for it.
type entry struct {
	kind *KindSpec
	obj  map[string]any // stored form, secrets included
	// script overrides kind.StatusScript when set (SetStatusScript).
	script []StatusStep
	// cursor is the index of the last applied status step; -1 right after a write.
	cursor int
	// deleting is set once a finalizer-mode DELETE has been accepted; getsLeft counts the GETs
	// that still return the object.
	deleting bool
	getsLeft int
}

func (e *entry) steps() []StatusStep {
	switch {
	case len(e.script) > 0:
		return e.script
	case len(e.kind.StatusScript) > 0:
		return e.kind.StatusScript
	default:
		return []StatusStep{{Status: "Success"}}
	}
}

// startOperation is what a write does to status: a new operation token is handed to the
// controller (status.processing.operationID) and the script restarts. lastReconcile is left as
// it was, so the object is not reconciled until the script's final step (D-WAIT,
// pkg/apis/core/v3/annotations.go:95-107: generation == status.lastReconcile.generation AND
// status.processing.operationID == status.lastReconcile.operationID).
// ASSUMPTION: a never-reconciled object reports status "Pending" (first value of the enum,
// pkg/apis/core/v3/status_types.go:69-70) and the server ignores client-sent status on writes.
func (s *Server) startOperation(e *entry, prevStatus map[string]any) {
	e.cursor = -1
	if !e.kind.HasStatus {
		delete(e.obj, "status")
		return
	}
	status := map[string]any{"status": "Pending"}
	if prevStatus != nil {
		status = prevStatus
	}
	status["processing"] = map[string]any{"operationID": s.nextID("op")}
	e.obj["status"] = status
}

// advance applies the next status step; called once per API GET of the object.
// Steps before the last are unreconciled; the last step records lastReconcile for the current
// generation and operation, i.e. the object becomes reconciled.
func (e *entry) advance() {
	if !e.kind.HasStatus {
		return
	}
	steps := e.steps()
	if e.cursor < len(steps)-1 {
		e.cursor++
	}
	step := steps[e.cursor]
	status := child(e.obj, "status", true)
	status["status"] = step.Status
	setOrDelete(status, "message", step.Message)
	setOrDelete(status, "detail", step.Detail)
	if e.cursor == len(steps)-1 {
		op := str(child(status, "processing", true), "operationID")
		status["lastReconcile"] = map[string]any{
			"generation":  generation(e.obj),
			"operationID": op,
			"time":        now().Format(time.RFC3339),
		}
	}
}

func setOrDelete(m map[string]any, key, val string) {
	if val == "" {
		delete(m, key)
		return
	}
	m[key] = val
}

// SetStatusScript replaces the status progression for one object and starts a new operation,
// so subsequent GETs walk the given steps from the first one.
func (s *Server) SetStatusScript(ref Ref, steps []StatusStep) {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.mustEntry(ref)
	e.script = append([]StatusStep(nil), steps...)
	s.startOperation(e, child(e.obj, "status", false))
}
