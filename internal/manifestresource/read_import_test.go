package manifestresource

import (
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/client"
	"github.com/appvia/terraform-provider-wayfinder/internal/engine"
	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRead_whenGenerationGatesFullImportedView(t *testing.T) {
	for _, imported := range []bool{true, false} {
		t.Run(map[bool]string{true: "import", false: "refresh"}[imported], func(t *testing.T) {
			// Given a partial manifest and a server-returned optional field.
			e := newTestEnv(t)
			e.fake.Seed(e.ref(t, "Pipeline", "ws", "dev", "build"), map[string]any{
				"apiVersion": "core.appvia.io/v3", "kind": "Pipeline", "spec": map[string]any{"trigger": "manual"},
			})
			r := newResource(e.m, engine.DefaultPoll())
			r.client = &client.Client{BaseURL: e.fake.URL, Tenant: "acme", TokenSource: testToken(fakeapi.DefaultToken)}
			var schemaResp resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(t.Context()), nil)}
			for key, value := range map[string]string{
				"id":       "core.appvia.io/Pipeline/ws/dev/build",
				"manifest": "apiVersion: core/v3\nkind: Pipeline\nmetadata: {name: build, workspace: ws, environment: dev}\nspec: {}\n",
			} {
				if d := state.SetAttribute(t.Context(), path.Root(key), value); d.HasError() {
					t.Fatal(d)
				}
			}
			resp := resource.ReadResponse{State: state}
			f := reflect.ValueOf(&resp).Elem().FieldByName("Private")
			f.Set(reflect.New(f.Type().Elem()))
			if !imported {
				if d := resp.Private.SetKey(t.Context(), generationKey, []byte("1")); d.HasError() {
					t.Fatal(d)
				}
			}
			// When reading with or without an existing generation.
			r.Read(t.Context(), resource.ReadRequest{State: state, Private: resp.Private}, &resp)
			// Then only import keeps the server's unconfigured optional field.
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var js jsontypes.Normalized
			if d := resp.State.GetAttribute(t.Context(), path.Root("object_json"), &js); d.HasError() {
				t.Fatal(d)
			}
			obj, err := decodeJSON(js.ValueString())
			if err != nil {
				t.Fatal(err)
			}
			_, present := obj["spec"].(map[string]any)["trigger"]
			if present != imported {
				t.Fatalf("trigger present = %v, want %v", present, imported)
			}
		})
	}
}

func TestImportedView_whenServerStampedMetadata(t *testing.T) {
	// Given server-stamped and user-set annotations.
	obj := map[string]any{"metadata": map[string]any{
		"annotations": map[string]any{"appvia.io/createdBy": "me", "appvia.io/lastUpdated": "now", "team": "x"},
		"labels":      map[string]any{"env": "dev"},
	}}
	// When building the imported view.
	got := importedView(obj)["metadata"].(map[string]any)
	// Then only user-set keys remain, and the source object is untouched.
	if !reflect.DeepEqual(got["annotations"], map[string]any{"team": "x"}) || !reflect.DeepEqual(got["labels"], map[string]any{"env": "dev"}) {
		t.Fatalf("metadata = %v", got)
	}
	if len(obj["metadata"].(map[string]any)["annotations"].(map[string]any)) != 3 {
		t.Fatal("source mutated")
	}
}
