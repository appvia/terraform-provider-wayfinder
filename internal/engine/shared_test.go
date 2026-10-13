package engine

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestWaiter_whenObjectReconcilesThenIsDeleted(t *testing.T) {
	// Given a stored object whose status script needs two GETs to reach Success.
	e := newTestEnv(t, "Workspace")
	ref := Reference(e.k, convert.Identity{Name: "example"})
	e.fake.Seed(e.ref("", "", ""), map[string]any{})
	e.fake.SetStatusScript(e.ref("", "", ""), []fakeapi.StatusStep{{Status: "Pending"}, {Status: "Success"}})
	w := Waiter{Client: e.c, Poll: Poll{Initial: time.Millisecond, Maximum: time.Millisecond}, Target: ErrorTarget{Kind: e.k.Kind, ID: "example", Address: "wayfinder_workspace.<name>"}}
	// When waiting for readiness, deleting, and waiting for removal.
	obj, d := w.Ready(t.Context(), ref)
	assertDiagnostics(t, d)
	if err := e.c.Delete(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	d = w.Gone(t.Context(), ref)
	// Then readiness returns the reconciled object and removal completes.
	assertDiagnostics(t, d)
	if g, err := Generation(obj); err != nil || g != 1 {
		t.Fatalf("generation = %d, %v", g, err)
	}
	if n := e.fake.CountRequests(http.MethodGet, ""); n < 3 {
		t.Fatalf("GET = %d, want >= 3", n)
	}
}

func TestAddAPIError_whenModifiedUsesTargetAddressAndID(t *testing.T) {
	// Given a stale-generation error for a caller-defined target.
	var d diag.Diagnostics
	// When reporting it.
	AddAPIError(&d, client.ErrModified, ErrorTarget{Kind: "Pipeline", ID: "core/Pipeline/ws/dev/p", Address: "wayfinder_manifest.<name>"})
	// Then the re-import instructions name the target address and ID.
	if len(d) != 1 || !strings.Contains(d[0].Detail(), "terraform import wayfinder_manifest.<name> core/Pipeline/ws/dev/p") {
		t.Fatalf("diagnostics = %v", d)
	}
}
