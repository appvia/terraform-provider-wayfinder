package convert

import (
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestAttributePath_whenNestedValidationField(t *testing.T) {
	// Given an API path with a list index and a reserved-name collision.
	m, err := NewMapping(manifest.Kind{Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"name": {Type: "string"}, "rules": {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"apiKey": {Type: "string", Sensitive: true}}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		json string
		want path.Path
	}{
		{"spec.name", path.Root("spec_name")},
		{"spec.rules[0].apiKey", path.Root("rules").AtListIndex(0).AtName("api_key_wo")},
		{"metadata.labels.team", path.Root("labels").AtMapKey("team")},
	} {
		// When mapping a server validation field back to Terraform.
		got, ok := m.AttributePath(tc.json)
		// Then the diagnostic is attached to the actual configured attribute.
		if !ok || !got.Equal(tc.want) {
			t.Fatalf("%s = %s (%t), want %s", tc.json, got, ok, tc.want)
		}
	}
}
