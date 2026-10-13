package convert_test

import (
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestFromObject_whenPopulateAllReadsGoldenOptionalFields(t *testing.T) {
	for _, tc := range []struct {
		kind, spec, attr, want string
	}{
		{"Integration", `{"endpoint":"https://example","timeoutSeconds":30,"region":"eu","lastSyncedAt":"today"}`, "timeout_seconds", `30`},
		{"Blueprint", `{"source":{"repository":"https://git.example","ref":"main"},"inputs":[{"name":"x","type":"string","required":false}]}`, "source", `{"repository":"https://git.example","ref":"main"}`},
		{"Workspace", `{"displayName":"Platform","cloudProvider":"aws","quota":{"maxEnvironments":5,"maxStorageGib":10.5}}`, "quota", `{"max_environments":5,"max_storage_gib":10.5}`},
		{"Pipeline", `{"trigger":"manual","steps":[{"name":"build","command":"make"}]}`, "steps", `[{"name":"build","command":"make"}]`},
	} {
		for _, populate := range []bool{false, true} {
			t.Run(tc.kind+map[bool]string{false: "/default", true: "/populate"}[populate], func(t *testing.T) {
				// Given an identity-only prior and server-returned optional fields.
				m := mapping(t, tc.kind)
				x := object(t, `{"metadata":{"name":"example","labels":{"server":"yes"},"annotations":{"controller":"yes"}},"spec":`+tc.spec+`}`)
				prior := config(t, m, nil)
				// When reading with the explicit import option (or the existing API).
				var got tftypes.Value
				var err error
				if populate {
					got, err = convert.FromObjectOpts(m, x, prior, convert.FromOpts{PopulateAll: true})
				} else {
					got, err = convert.FromObject(m, x, prior)
				}
				// Then import hydrates optional fields and all metadata keys; refresh filters them.
				if err != nil {
					t.Fatal(err)
				}
				a := values(t, got)
				if !a["timeouts"].IsNull() {
					t.Fatal("import must not populate timeouts")
				}
				if ready, exists := a["wait_for_ready"]; exists {
					wantReady := tftypes.NewValue(tftypes.Bool, nil)
					if populate {
						wantReady = tftypes.NewValue(tftypes.Bool, true)
					}
					if !ready.Equal(wantReady) {
						t.Fatalf("wait_for_ready = %s, want %s", ready, wantReady)
					}
				}
				want := tftypes.NewValue(m.TFType().AttributeTypes[tc.attr], nil)
				if populate {
					want, err = fixtureValue(want.Type(), tc.want)
					if err != nil {
						t.Fatal(err)
					}
				} else if tc.kind == "Blueprint" {
					want, err = fixtureValue(want.Type(), `{"repository":"https://git.example","ref":null}`)
					if err != nil {
						t.Fatal(err)
					}
				}
				if !a[tc.attr].Equal(want) {
					t.Fatalf("%s = %s, want %s", tc.attr, a[tc.attr], want)
				}
				for _, key := range []string{"labels", "annotations"} {
					if a[key].IsNull() == populate {
						t.Fatalf("%s null = %v", key, a[key].IsNull())
					}
				}
			})
		}
	}
}

func TestFromObject_whenPopulateAllClearsWriteOnlyPairs(t *testing.T) {
	// Given secrets returned by the server and prior version companions.
	m := mapping(t, "Integration")
	prior := config(t, m, map[string]any{"api_key_wo_version": "1", "headers_wo_version": "2"})
	x := object(t, `{"spec":{"apiKey":"LEAK","headers":{"X":"LEAK"},"lastSyncedAt":"today"}}`)
	// When import hydrates every manifest field.
	got, err := convert.FromObjectOpts(m, x, prior, convert.FromOpts{PopulateAll: true})
	// Then write-only values and companions are null while computed fields still populate.
	if err != nil {
		t.Fatal(err)
	}
	a := values(t, got)
	for _, key := range []string{"api_key_wo", "api_key_wo_version", "headers_wo", "headers_wo_version"} {
		if !a[key].IsNull() {
			t.Fatalf("%s should be null", key)
		}
	}
	if !a["last_synced_at"].Equal(tftypes.NewValue(tftypes.String, "today")) {
		t.Fatal("computed value missing")
	}
}

func TestFromObject_whenPopulateAllRecursesIntoListsAndMaps(t *testing.T) {
	// Given unconfigured list/map objects with nested optional fields and secrets.
	m := nestedMapping(t)
	x := object(t, `{"spec":{"rules":[{"displayName":"rule","payload":{"x":1},"token":"LEAK"}],"entries":{"key":{"displayName":"entry","extra":"default","token":"LEAK"}}}}`)
	// When import reads the complete manifest-described shape.
	got, err := convert.FromObjectOpts(m, x, config(t, m, nil), convert.FromOpts{PopulateAll: true})
	// Then nested fields populate recursively, without secrets or version companions.
	if err != nil {
		t.Fatal(err)
	}
	for key, text := range map[string]string{
		"rules":   `[{"display_name":"rule","payload_json":"{\"x\":1}","token_wo":null,"token_wo_version":null}]`,
		"entries": `{"key":{"display_name":"entry","extra":"default","payload_json":null,"token_wo":null,"token_wo_version":null}}`,
	} {
		want, err := fixtureValue(m.TFType().AttributeTypes[key], text)
		if err != nil {
			t.Fatal(err)
		}
		if !values(t, got)[key].Equal(want) {
			t.Fatalf("%s = %s, want %s", key, values(t, got)[key], want)
		}
	}
}
