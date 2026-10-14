package manifestresource

import (
	"reflect"
	"strings"
	"testing"
)

const woLiteral = "s3cr3t-literal"

func TestParseWriteOnly_whenValid(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       map[string]any
	}{
		{
			name: "yaml keeps null list placeholders",
			text: "spec:\n  rules:\n    - null\n    - token: abc\n",
			want: map[string]any{"spec": map[string]any{"rules": []any{nil, map[string]any{"token": "abc"}}}},
		},
		{
			name: "json object",
			text: `{"spec":{"security":{"secret":"abc"}}}`,
			want: map[string]any{"spec": map[string]any{"security": map[string]any{"secret": "abc"}}},
		},
		{
			name: "empty object is allowed",
			text: "{}",
			want: map[string]any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given manifest_wo text that is one valid YAML or JSON object.
			// When parsing it.
			got, problems := parseWriteOnly(tc.text)
			// Then it returns the object with no problems.
			if len(problems) != 0 {
				t.Fatalf("unexpected problems: %+v", problems)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestParseWriteOnly_whenInvalid(t *testing.T) {
	for _, tc := range []struct {
		name, text, wantDetail string
	}{
		{
			name:       "multiple yaml documents",
			text:       "spec: {}\n---\nspec: {}\n",
			wantDetail: "manifest_wo holds 2 YAML documents; it must be one object.",
		},
		{
			name:       "scalar",
			text:       "just-a-string",
			wantDetail: "manifest_wo must be one YAML or JSON object.",
		},
		{
			name:       "list",
			text:       "- a\n- b\n",
			wantDetail: "manifest_wo must be one YAML or JSON object.",
		},
		{
			name:       "null document",
			text:       "null",
			wantDetail: "manifest_wo must be one YAML or JSON object.",
		},
		{
			name:       "forbidden apiVersion",
			text:       "apiVersion: core.appvia.io/v3\n",
			wantDetail: "manifest_wo must not set apiVersion; set it in manifest.",
		},
		{
			name:       "forbidden kind",
			text:       "kind: Group\n",
			wantDetail: "manifest_wo must not set kind; set it in manifest.",
		},
		{
			name:       "forbidden metadata",
			text:       "metadata: {name: x}\n",
			wantDetail: "manifest_wo must not set metadata; set it in manifest.",
		},
		{
			name:       "forbidden status",
			text:       "status: {}\n",
			wantDetail: "manifest_wo must not set status; set it in manifest.",
		},
		{
			name:       "malformed yaml with secret",
			text:       "spec: {token: " + woLiteral + "\n",
			wantDetail: "manifest_wo is not valid YAML or JSON",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given manifest_wo text that is not one valid object.
			// When parsing it.
			_, problems := parseWriteOnly(tc.text)
			// Then exactly the expected problem is reported and never echoes a secret value.
			if len(problems) == 0 {
				t.Fatalf("expected a problem for %q", tc.name)
			}
			found := false
			for _, p := range problems {
				if p.attr != "manifest_wo" {
					t.Fatalf("problem attr = %q, want manifest_wo", p.attr)
				}
				if strings.Contains(p.detail, woLiteral) {
					t.Fatalf("problem detail echoes the write-only value: %q", p.detail)
				}
				if strings.HasPrefix(p.detail, tc.wantDetail) {
					found = true
				}
			}
			if !found {
				t.Fatalf("no problem with detail %q in %+v", tc.wantDetail, problems)
			}
		})
	}
}
