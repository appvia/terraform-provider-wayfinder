package engine

import (
	"net/http"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestCreate_whenAlreadyExistsDoesNotRetry(t *testing.T) {
	// Given an object already managed elsewhere.
	e := newTestEnv(t, "Workspace")
	e.fake.Seed(e.ref("", "", ""), map[string]any{})
	// When Terraform attempts create.
	resp := createDirect(t, e, rawValue(t, e, nil))
	// Then one POST fails with actionable import instructions.
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), "terraform import wayfinder_workspace.<name> example") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if n := e.fake.CountRequests(http.MethodPost, ""); n != 1 {
		t.Fatalf("POST = %d", n)
	}
}

func TestValidation_whenServerReturnsMultipleFields(t *testing.T) {
	// Given field errors from the real HTTP fake.
	e := newTestEnv(t, "Integration")
	k := kindSpec(e.k)
	k.Validate = func(fakeapi.Operation, map[string]any) []fakeapi.FieldError {
		return []fakeapi.FieldError{{Field: "spec.endpoint", Message: "endpoint invalid"}, {Field: "spec.apiKey", Message: "key invalid"}}
	}
	e.fake.RegisterKind(k)
	// When creating.
	resp := createDirect(t, e, rawValue(t, e, map[string]any{"workspace": "ws"}))
	// Then each field has its own diagnostic at the mapped Terraform path.
	if len(resp.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	for i, want := range []path.Path{path.Root("endpoint"), path.Root("api_key_wo")} {
		d, ok := resp.Diagnostics[i].(diag.DiagnosticWithPath)
		if !ok || !d.Path().Equal(want) {
			t.Fatalf("field %d = %v", i, resp.Diagnostics[i])
		}
	}
}

func TestAPIDiagnostics_whenServerMessageMustRemainVerbatim(t *testing.T) {
	// Given upstream 5xx and dependency-conflict messages.
	e := newTestEnv(t, "Workspace")
	r := directResource(e)
	for _, code := range []int{500, 409} {
		var d diag.Diagnostics
		// When reporting the typed upstream error.
		r.apiDiagnostics(&d, &client.APIError{StatusCode: code, Message: "server text exactly", Detail: "not added"}, "example")
		// Then the provider neither decorates nor paraphrases it.
		if len(d) != 1 || d[0].Detail() != "server text exactly" {
			t.Fatalf("diagnostics = %v", d)
		}
	}
}

func TestValidation_whenServerReportsBareAndUnknownFields(t *testing.T) {
	// Given a bare identity field error and an unknown field error.
	e := newTestEnv(t, "Integration")
	k := kindSpec(e.k)
	k.Validate = func(fakeapi.Operation, map[string]any) []fakeapi.FieldError {
		return []fakeapi.FieldError{{Field: "name", Message: "name invalid"}, {Field: "mystery", Message: "mystery invalid"}}
	}
	e.fake.RegisterKind(k)
	// When creating.
	resp := createDirect(t, e, rawValue(t, e, map[string]any{"workspace": "ws"}))
	// Then name is attached to the name attribute; the unknown field keeps its message with no path.
	if len(resp.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if d, ok := resp.Diagnostics[0].(diag.DiagnosticWithPath); !ok || !d.Path().Equal(path.Root("name")) || d.Detail() != "name invalid" {
		t.Fatalf("name diagnostic = %v", resp.Diagnostics[0])
	}
	if _, ok := resp.Diagnostics[1].(diag.DiagnosticWithPath); ok || resp.Diagnostics[1].Detail() != "mystery invalid" {
		t.Fatalf("unknown diagnostic = %v", resp.Diagnostics[1])
	}
}
