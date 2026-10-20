// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package fakeapi_test

import (
	"net/http"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
)

// reconciled mirrors corev3.IsReconciled (pkg/apis/core/v3/annotations.go:95-107) as D-WAIT uses it.
func reconciled(obj map[string]any) bool {
	gen := dig(obj, "metadata", "generation")
	lrGen := dig(obj, "status", "lastReconcile", "generation")
	opProc, _ := dig(obj, "status", "processing", "operationID").(string)
	opDone, _ := dig(obj, "status", "lastReconcile", "operationID").(string)

	return gen == lrGen && opProc == opDone
}

func pollStatuses(t *testing.T, s *fakeapi.Server, path string, n int) []map[string]any {
	t.Helper()

	out := make([]map[string]any, 0, n)
	for range n {
		r := call(t, s, http.MethodGet, path, nil)
		wantCode(t, r, http.StatusOK)
		out = append(out, r.body)
	}

	return out
}

func TestStatus_script_progresses_pending_twice_then_success_and_reconciles_last(t *testing.T) {
	k := identityKind()
	k.StatusScript = []fakeapi.StatusStep{{Status: "Pending"}, {Status: "Pending"}, {Status: "Success", Message: "ready"}}
	s := newServer(t, k)
	created := call(t, s, http.MethodPost, identityCollection, identityObject("id1"))

	polls := pollStatuses(t, s, identityCollection+"/id1", 4)

	if reconciled(created.body) {
		t.Error("create response already reconciled")
	}

	want := []string{"Pending", "Pending", "Success", "Success"}
	for i, p := range polls {
		if st := dig(p, "status", "status"); st != want[i] {
			t.Errorf("poll %d status = %v, want %s", i, st, want[i])
		}

		if reconciled(p) != (i >= 2) {
			t.Errorf("poll %d reconciled = %v", i, reconciled(p))
		}
	}

	if dig(polls[2], "status", "message") != "ready" {
		t.Errorf("message = %v", dig(polls[2], "status", "message"))
	}
}

func TestStatus_script_restarts_after_update_with_new_operation(t *testing.T) {
	k := identityKind()
	k.StatusScript = []fakeapi.StatusStep{{Status: "Pending"}, {Status: "Success"}}
	s := newServer(t, k)
	call(t, s, http.MethodPost, identityCollection, identityObject("id1"))
	done := pollStatuses(t, s, identityCollection+"/id1", 2)[1]

	upd := call(t, s, http.MethodPut, identityCollection+"/id1", done)
	polls := pollStatuses(t, s, identityCollection+"/id1", 2)

	if reconciled(upd.body) || reconciled(polls[0]) || !reconciled(polls[1]) {
		t.Errorf("reconciled after update = %v %v %v", reconciled(upd.body), reconciled(polls[0]), reconciled(polls[1]))
	}

	if dig(polls[1], "status", "lastReconcile", "operationID") == dig(done, "status", "lastReconcile", "operationID") {
		t.Error("update did not start a new operation")
	}
}

func TestStatus_script_terminal_variants_carry_message_and_detail(t *testing.T) {
	for _, st := range []string{"Failure", "ApprovalRequired", "ActionRequired"} {
		t.Run(st, func(t *testing.T) {
			s := newServer(t, identityKind())
			call(t, s, http.MethodPost, identityCollection, identityObject("id1"))
			s.SetStatusScript(identityRef("id1"), []fakeapi.StatusStep{{Status: st, Message: "m", Detail: "d"}})

			got := pollStatuses(t, s, identityCollection+"/id1", 1)[0]

			if dig(got, "status", "status") != st || dig(got, "status", "message") != "m" || dig(got, "status", "detail") != "d" || !reconciled(got) {
				t.Errorf("status = %v", got["status"])
			}
		})
	}
}

func TestStatus_absent_for_kinds_without_status(t *testing.T) {
	k := identityKind()
	k.HasStatus = false
	s := newServer(t, k)
	call(t, s, http.MethodPost, identityCollection, identityObject("id1"))

	got := pollStatuses(t, s, identityCollection+"/id1", 1)[0]

	if _, ok := got["status"]; ok {
		t.Errorf("status present: %v", got["status"])
	}
}
