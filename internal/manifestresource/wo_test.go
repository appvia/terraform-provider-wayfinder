package manifestresource

import (
	"reflect"
	"strings"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
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

func TestMergeWriteOnly_whenMerging(t *testing.T) {
	m := testManifest(t)
	integration := findKind(t, m, "Integration")
	externalIdentity := findKind(t, m, "ExternalIdentity")
	pipeline := findKind(t, m, "Pipeline")
	for _, tc := range []struct {
		name           string
		kind           manifest.Kind
		base, wo, want map[string]any
	}{
		{
			name: "nested object merge", kind: externalIdentity,
			base: map[string]any{"type": "bearerToken", "bearerToken": map[string]any{}},
			wo:   map[string]any{"bearerToken": map[string]any{"token": woLiteral}},
			want: map[string]any{"type": "bearerToken", "bearerToken": map[string]any{"token": woLiteral}},
		},
		{
			name: "list per index with null placeholder", kind: pipeline,
			base: map[string]any{"steps": []any{map[string]any{"name": "build"}, map[string]any{"name": "deploy"}}},
			wo:   map[string]any{"steps": []any{nil, map[string]any{"command": woLiteral}}},
			want: map[string]any{"steps": []any{map[string]any{"name": "build"}, map[string]any{"name": "deploy", "command": woLiteral}}},
		},
		{
			name: "blank secret placeholder", kind: externalIdentity,
			base: map[string]any{"bearerToken": map[string]any{"token": ""}},
			wo:   map[string]any{"bearerToken": map[string]any{"token": woLiteral}},
			want: map[string]any{"bearerToken": map[string]any{"token": woLiteral}},
		},
		{
			name: "secret map with blank keys", kind: integration,
			base: map[string]any{"headers": map[string]any{"X-A": ""}},
			wo:   map[string]any{"headers": map[string]any{"X-A": "a", "X-B": "b"}},
			want: map[string]any{"headers": map[string]any{"X-A": "a", "X-B": "b"}},
		},
		{
			name: "spec-less kind blank placeholder", kind: manifest.Kind{Kind: "Thing"},
			base: map[string]any{"credentials": ""},
			wo:   map[string]any{"credentials": map[string]any{"tokens": []any{woLiteral}}},
			want: map[string]any{"credentials": map[string]any{"tokens": []any{woLiteral}}},
		},
		{
			name: "null overlay keeps existing and absent keys", kind: integration,
			base: map[string]any{"endpoint": "https://example.test"},
			wo:   map[string]any{"endpoint": nil, "apiKey": nil},
			want: map[string]any{"endpoint": "https://example.test"},
		},
		{
			name: "blank secret map replaced by scalar", kind: integration,
			base: map[string]any{"headers": map[string]any{"X-A": ""}},
			wo:   map[string]any{"headers": woLiteral},
			want: map[string]any{"headers": woLiteral},
		},
		{
			name: "blank secret list replaced by map", kind: integration,
			base: map[string]any{"headers": []any{"", nil, map[string]any{}}},
			wo:   map[string]any{"headers": map[string]any{"X-A": woLiteral}},
			want: map[string]any{"headers": map[string]any{"X-A": woLiteral}},
		},
		{
			name: "null base secret placeholder", kind: integration,
			base: map[string]any{"apiKey": nil},
			wo:   map[string]any{"apiKey": woLiteral},
			want: map[string]any{"apiKey": woLiteral},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given independent base and write-only spec trees.
			base := map[string]any{"spec": tc.base}
			wo := map[string]any{"spec": tc.wo}
			baseBefore, woBefore := deepCopy(base), deepCopy(wo)
			// When merging write-only values.
			got, problems := mergeWriteOnly(tc.kind, base, wo)
			// Then the result matches the expected tree without mutating either input.
			if len(problems) != 0 {
				t.Fatalf("unexpected problems: %+v", problems)
			}
			if want := (map[string]any{"spec": tc.want}); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			if !reflect.DeepEqual(base, baseBefore) || !reflect.DeepEqual(wo, woBefore) {
				t.Fatal("merge mutated an input")
			}
			t.Log("merged map equals expected; both inputs unchanged; zero problems")
		})
	}
}

func TestMergeWriteOnly_whenValuesConflict(t *testing.T) {
	k := findKind(t, testManifest(t), "Integration")
	for _, tc := range []struct {
		name, key, path string
		base, wo        any
	}{
		{"non-secret value", "endpoint", "spec.endpoint", "https://example.test", woLiteral},
		{"blank non-secret placeholder", "endpoint", "spec.endpoint", "", woLiteral},
		{"null non-secret placeholder", "endpoint", "spec.endpoint", nil, woLiteral},
		{"scalar versus map", "endpoint", "spec.endpoint", "https://example.test", map[string]any{"token": woLiteral}},
		{"map versus list", "endpoint", "spec.endpoint", map[string]any{}, []any{woLiteral}},
		{"nonblank secret", "apiKey", "spec.apiKey", "existing", woLiteral},
		{"equal secret values", "apiKey", "spec.apiKey", woLiteral, woLiteral},
		{"false is set", "apiKey", "spec.apiKey", false, woLiteral},
		{"zero is set", "apiKey", "spec.apiKey", 0, woLiteral},
		{"quoted map key", "headers", `spec.headers["X.Api-Key"]`, map[string]any{"X.Api-Key": "existing"}, map[string]any{"X.Api-Key": woLiteral}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given overlapping values that cannot use the blank-secret exception.
			base := map[string]any{"spec": map[string]any{tc.key: tc.base}}
			wo := map[string]any{"spec": map[string]any{tc.key: tc.wo}}
			baseBefore, woBefore := deepCopy(base), deepCopy(wo)
			// When merging them.
			got, problems := mergeWriteOnly(k, base, wo)
			// Then exactly one conflict names the canonical path without echoing values.
			if len(problems) != 1 {
				t.Fatalf("got %d problems, want 1: %+v", len(problems), problems)
			}
			p := problems[0]
			if p.attr != "manifest_wo" || p.summary != "Conflicting manifest_wo value" || !strings.HasPrefix(p.detail, tc.path+" is set in both ") {
				t.Fatalf("unexpected problem: %+v", p)
			}
			if strings.Contains(p.attr+p.summary+p.detail, woLiteral) {
				t.Fatal("problem echoes the write-only literal")
			}
			if !reflect.DeepEqual(got, baseBefore) || !reflect.DeepEqual(base, baseBefore) || !reflect.DeepEqual(wo, woBefore) {
				t.Fatal("conflict changed the base or either input")
			}
			t.Logf("exactly one problem: %s", p.detail)
		})
	}
}

func TestMergeWriteOnly_whenListLengthsDiffer(t *testing.T) {
	k := findKind(t, testManifest(t), "Pipeline")
	for _, tc := range []struct {
		name, counts string
		base, wo     []any
	}{
		{"overlay longer", "2 items in manifest_wo but 1 in manifest", []any{map[string]any{"name": "build"}}, []any{nil, woLiteral}},
		{"overlay shorter", "1 items in manifest_wo but 2 in manifest", []any{nil, nil}, []any{woLiteral}},
		{"empty base", "1 items in manifest_wo but 0 in manifest", []any{}, []any{woLiteral}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given two lists of different lengths at the same path.
			base := map[string]any{"spec": map[string]any{"steps": tc.base}}
			wo := map[string]any{"spec": map[string]any{"steps": tc.wo}}
			baseBefore, woBefore := deepCopy(base), deepCopy(wo)
			// When merging them.
			got, problems := mergeWriteOnly(k, base, wo)
			// Then exactly one problem names the path and counts, never the values.
			if len(problems) != 1 {
				t.Fatalf("got %d problems, want 1: %+v", len(problems), problems)
			}
			p := problems[0]
			if p.attr != "manifest_wo" || p.summary != "Invalid manifest_wo" || !strings.HasPrefix(p.detail, "spec.steps has "+tc.counts+";") {
				t.Fatalf("unexpected problem: %+v", p)
			}
			if strings.Contains(p.attr+p.summary+p.detail, woLiteral) {
				t.Fatal("problem echoes the write-only literal")
			}
			if !reflect.DeepEqual(got, baseBefore) || !reflect.DeepEqual(base, baseBefore) || !reflect.DeepEqual(wo, woBefore) {
				t.Fatal("length mismatch changed the base or either input")
			}
			t.Logf("exactly one problem: %s", p.detail)
		})
	}
}

func TestMergeWriteOnly_whenInputsMustRemainIndependent(t *testing.T) {
	// Given nested base and write-only maps and lists, including a blank replacement.
	base := map[string]any{"spec": map[string]any{
		"keep":  map[string]any{"name": "public"},
		"items": []any{map[string]any{"name": "first"}, nil},
		"token": "",
	}}
	wo := map[string]any{"spec": map[string]any{
		"extra": map[string]any{"tokens": []any{woLiteral}},
		"items": []any{nil, map[string]any{"token": woLiteral}},
		"token": map[string]any{"value": woLiteral},
	}}
	baseBefore, woBefore := deepCopy(base), deepCopy(wo)
	// When merging them.
	got, problems := mergeWriteOnly(manifest.Kind{Kind: "Thing"}, base, wo)
	// Then modifying every copied branch of the result leaves both inputs unchanged.
	if len(problems) != 0 {
		t.Fatalf("unexpected problems: %+v", problems)
	}
	spec := got["spec"].(map[string]any)
	spec["keep"].(map[string]any)["name"] = "changed"
	spec["extra"].(map[string]any)["tokens"].([]any)[0] = "changed"
	spec["items"].([]any)[0].(map[string]any)["name"] = "changed"
	spec["items"].([]any)[1].(map[string]any)["token"] = "changed"
	spec["token"].(map[string]any)["value"] = "changed"
	if !reflect.DeepEqual(base, baseBefore) || !reflect.DeepEqual(wo, woBefore) {
		t.Fatal("merged result aliases an input")
	}
	t.Log("result map and list mutations leave both input snapshots equal")
}

func TestMergeWriteOnly_whenConflictsNeedDeterministicOrder(t *testing.T) {
	// Given conflicts whose map insertion order differs from their canonical path order.
	base := map[string]any{"spec": map[string]any{"z": "public", "a": []any{"public"}}}
	wo := map[string]any{"spec": map[string]any{"z": woLiteral, "a": []any{woLiteral}}}
	// When merging them.
	_, problems := mergeWriteOnly(manifest.Kind{Kind: "Thing"}, base, wo)
	// Then problems follow sorted keys and preserve list indexes.
	if len(problems) != 2 {
		t.Fatalf("got %d problems, want 2: %+v", len(problems), problems)
	}
	for i, path := range []string{"spec.a[0]", "spec.z"} {
		if !strings.HasPrefix(problems[i].detail, path+" ") || strings.Contains(problems[i].detail, woLiteral) {
			t.Fatalf("unexpected problem at index %d: %+v", i, problems[i])
		}
	}
	t.Log("two problems in canonical path order: spec.a[0], spec.z; no write-only literal")
}

func TestIsSecretPath_whenWalkingFields(t *testing.T) {
	m := testManifest(t)
	integration := findKind(t, m, "Integration")
	externalIdentity := findKind(t, m, "ExternalIdentity")
	schema := manifest.Kind{Spec: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{
		"rules":  {Type: "array", Items: &manifest.Field{Type: "object", Properties: map[string]*manifest.Field{"token": {Type: "string", Sensitive: true}}}},
		"tokens": {Type: "map", Values: &manifest.Field{Type: "string", Sensitive: true}},
		"opaque": {Type: "json"},
	}}}
	for _, tc := range []struct {
		name string
		kind manifest.Kind
		path string
		want bool
	}{
		{"secret scalar", integration, "spec.apiKey", true},
		{"nested secret", externalIdentity, "spec.bearerToken.token", true},
		{"non-secret scalar", integration, "spec.endpoint", false},
		{"non-secret parent", externalIdentity, "spec.bearerToken", false},
		{"secret map itself", integration, "spec.headers", true},
		{"secret map ancestor", integration, `spec.headers["X.Api-Key"]`, true},
		{"spec-less kind", manifest.Kind{Kind: "Thing"}, "other.token", true},
		{"outside spec", integration, "metadata.token", false},
		{"empty path", integration, "", false},
		{"unknown property", integration, "spec.unknown", false},
		{"object index mismatch", integration, "spec[0]", false},
		{"scalar child mismatch", integration, "spec.endpoint.token", false},
		{"json child unknown", integration, "spec.options.token", false},
		{"non-secret json itself", schema, "spec.opaque", false},
		{"array item secret", schema, "spec.rules[0].token", true},
		{"array key mismatch", schema, "spec.rules.token", false},
		{"unknown array item property", schema, "spec.rules[0].unknown", false},
		{"map value secret", schema, "spec.tokens.any", true},
		{"map index mismatch", schema, "spec.tokens[0]", false},
		{"starting spec sensitive", manifest.Kind{Spec: &manifest.Field{Type: "object", Sensitive: true}}, "spec", true},
		{"starting spec sensitive ancestor", manifest.Kind{Spec: &manifest.Field{Type: "object", Sensitive: true}}, "spec.any.token", true},
		{"sensitive array ancestor", manifest.Kind{Spec: &manifest.Field{Type: "array", Sensitive: true}}, "spec[0].token", true},
		{"sensitive array item", manifest.Kind{Spec: &manifest.Field{Type: "array", Items: &manifest.Field{Type: "string", Sensitive: true}}}, "spec[0]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a schema and a canonical path into it.
			var p woPath
			if tc.path != "" {
				var err error
				p, err = parseWOPath(tc.path)
				if err != nil {
					t.Fatal(err)
				}
			}
			// When classifying the path.
			got := isSecretPath(tc.kind, p)
			// Then sensitivity follows self or ancestors, never unknown shape guesses.
			if got != tc.want {
				t.Fatalf("isSecretPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}
