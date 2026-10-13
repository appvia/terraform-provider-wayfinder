package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestWait_whenGenerationOrOperationNotReconciled(t *testing.T) {
	for _, mismatch := range []string{"generation", "operation", "empty operations"} {
		t.Run(mismatch, func(t *testing.T) {
			// Given Success whose acknowledgement is initially stale.
			e := newTestEnv(t, "Workspace")
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				generation := 1
				processing, last := "new", "new"
				if mismatch == "empty operations" {
					processing, last = "", ""
				}
				if calls == 1 && mismatch == "generation" {
					generation = 0
				}
				if calls == 1 && mismatch == "operation" {
					last = "old"
				}
				if err := json.NewEncoder(w).Encode(map[string]any{"metadata": map[string]any{"name": "example", "generation": 1}, "status": map[string]any{"status": "Success", "processing": map[string]any{"operationID": processing}, "lastReconcile": map[string]any{"generation": generation, "operationID": last}}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			e.c.BaseURL = server.URL
			r := directResource(e)
			// When polling readiness.
			_, d := r.wait(t.Context(), reference(e.k, convert.Identity{Name: "example"}), false)
			// Then stale acknowledgements need another GET; empty equal tokens succeed.
			assertDiagnostics(t, d)
			want := 2
			if mismatch == "empty operations" {
				want = 1
			}
			if calls != want {
				t.Fatalf("GETs = %d, want %d", calls, want)
			}
		})
	}
}

func TestCreate_whenReadinessDisabledSkipsPolling(t *testing.T) {
	for _, manifestDisabled := range []bool{false, true} {
		// Given a Pending object and either config or manifest disabling readiness.
		e := newTestEnv(t, "Workspace")
		if manifestDisabled {
			e.k.WaitForReady = false
		}
		k := kindSpec(e.k)
		k.StatusScript = []fakeapi.StatusStep{{Status: "Pending"}}
		e.fake.RegisterKind(k)
		// When creating.
		resp := createDirect(t, e, rawValue(t, e, map[string]any{"wait_for_ready": false}))
		// Then creation does not poll or time out.
		assertDiagnostics(t, resp.Diagnostics)
		if n := e.fake.CountRequests(http.MethodGet, ""); n != 0 {
			t.Fatalf("GETs = %d", n)
		}
	}
}

func TestUpdate_whenServerCASConflictDoesNotRetry(t *testing.T) {
	// Given a baseline established by create and a race after the fresh GET.
	e := newTestEnv(t, "Workspace")
	v := rawValue(t, e, map[string]any{"display_name": "First", "cloud_provider": "aws"})
	created := createDirect(t, e, v)
	assertDiagnostics(t, created.Diagnostics)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			e.fake.ServeHTTP(w, req)
			e.fake.Mutate(e.ref("", "", ""), func(map[string]any) {})
			return
		}
		e.fake.ServeHTTP(w, req)
	}))
	defer server.Close()
	e.c.BaseURL = server.URL
	e.fake.ResetRequests()
	r := directResource(e)
	resp := resource.UpdateResponse{State: created.State, Private: created.Private}
	// When updating with a now-stale generation in the PUT body.
	r.Update(t.Context(), resource.UpdateRequest{State: created.State, Private: created.Private, Config: tfsdk.Config{Raw: v, Schema: r.schema}, Plan: tfsdk.Plan{Raw: v, Schema: r.schema}}, &resp)
	// Then the server conflict gets the drift diagnostic and exactly one PUT.
	if !resp.Diagnostics.HasError() || !modifiedRegex().MatchString(resp.Diagnostics[0].Detail()) {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if n := e.fake.CountRequests(http.MethodPut, ""); n != 1 {
		t.Fatalf("PUTs = %d", n)
	}
	t.Log("CAS race: PUT request count = 1; no retry")
}

func TestDelete_whenServerConflictOrMissing(t *testing.T) {
	for _, code := range []int{404, 409} {
		// Given a DELETE response with dependency conflict or absence.
		e := newTestEnv(t, "Workspace")
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.WriteHeader(code)
			if err := json.NewEncoder(w).Encode(map[string]string{"message": "dependency remains"}); err != nil {
				t.Error(err)
			}
		}))
		defer server.Close()
		e.c.BaseURL = server.URL
		r := directResource(e)
		resp := resource.DeleteResponse{}
		// When deleting.
		r.Delete(t.Context(), resource.DeleteRequest{State: tfsdk.State{Raw: rawValue(t, e, nil), Schema: r.schema}}, &resp)
		// Then absence succeeds, conflict preserves the message, and neither retries.
		if resp.Diagnostics.HasError() != (code == 409) || calls != 1 {
			t.Fatalf("code %d: %v, calls %d", code, resp.Diagnostics, calls)
		}
		if code == 409 && resp.Diagnostics[0].Detail() != "dependency remains" {
			t.Fatalf("message = %v", resp.Diagnostics)
		}
	}
}
