// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseManifest_whenAPIVersionUsesShortOrQualifiedGroup(t *testing.T) {
	m := testManifest(t)
	for _, tc := range []struct{ text, wantID string }{
		{"apiVersion: org/v3\nkind: Group\nmetadata: {name: developers, workspace: ws1}\n", "org/Group/ws1/developers"},
		{"apiVersion: org.appvia.io/v3\nkind: Group\nmetadata: {name: developers, workspace: ws1}\n", "org/Group/ws1/developers"},
		{`{"apiVersion":"core/v3","kind":"Workspace","metadata":{"name":"plat"}}`, "core.appvia.io/Workspace/plat"},
		{"apiVersion: core.appvia.io/v3\nkind: Blueprint\nmetadata: {name: bp, workspace: ws, version: \"1.2.3\"}\n", "core.appvia.io/Blueprint/ws/bp@1.2.3"},
	} {
		t.Run(tc.wantID, func(t *testing.T) {
			// Given a manifest naming its group by short token or fully-qualified group.
			// When parsing it.
			d, err := parseManifest(m, tc.text)
			// Then it resolves to the manifest kind and canonical apiVersion.
			if err != nil {
				t.Fatal(err)
			}

			if d.importID() != tc.wantID {
				t.Fatalf("id = %s", d.importID())
			}

			if d.object["apiVersion"] != d.kind.Group+"/"+d.kind.Version {
				t.Fatalf("apiVersion = %v", d.object["apiVersion"])
			}
		})
	}
}

func TestParseManifest_whenInvalid(t *testing.T) {
	m := testManifest(t)
	for _, tc := range []struct{ name, text, want string }{
		{"excluded", "apiVersion: core/v3\nkind: Run\nmetadata: {name: r}\n", "Kind Run cannot be managed with Terraform."},
		{"unknown", "apiVersion: core/v3\nkind: Gadget\nmetadata: {name: g}\n", "Kind Gadget is not supported by this provider version; upgrade the provider."},
		{"wrong version", "apiVersion: org/v2\nkind: Group\nmetadata: {name: g, workspace: w}\n", "use org/v3"},
		{"no apiVersion", "kind: Group\nmetadata: {name: g}\n", "<group>/<version>"},
		{"no name", "apiVersion: org/v3\nkind: Group\nmetadata: {workspace: w}\n", "metadata.name is required"},
		{"wrong scope", "apiVersion: org/v3\nkind: Group\nmetadata: {name: g}\n", "does not match kind Group"},
		{"two docs", "apiVersion: org/v3\nkind: Group\n---\napiVersion: org/v3\nkind: Group\n", "2 YAML documents"},
		{"list", "- a\n", "one object"},
		{"slash in name", "apiVersion: core/v3\nkind: Workspace\nmetadata: {name: a/b}\n", "without '/'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given an unusable manifest. When parsing it.
			_, err := parseManifest(m, tc.text)
			// Then the error says what is wrong.
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseID_whenGroupKindAndIdentity(t *testing.T) {
	m := testManifest(t)
	// Given import IDs. When parsing them. Then kind and identity are resolved or rejected.
	k, i, err := parseID(m, "core/Pipeline/ws/dev/p")
	if err != nil || k.Kind != "Pipeline" || i.Workspace != "ws" || i.Environment != "dev" || i.Name != "p" {
		t.Fatalf("got %v %v %v", k.Kind, i, err)
	}

	for _, bad := range []string{"plat", "core/Workspace", "core/Workspace/ws/plat", "core/Run/x"} {
		if _, _, err := parseID(m, bad); err == nil {
			t.Fatalf("%s accepted", bad)
		}
	}

	var ke *kindError
	if _, _, err := parseID(m, "core/Run/x"); !errors.As(err, &ke) {
		t.Fatalf("err = %v", err)
	}
}

func TestProject_whenServerAddsKeysAndElements(t *testing.T) {
	// Given a server object with defaults, status, extra labels and an extra list element.
	server := map[string]any{"metadata": map[string]any{"name": "a", "generation": 2.0, "labels": map[string]any{"team": "x", "added": "y"}},
		"spec": map[string]any{"list": []any{map[string]any{"a": 1.0, "def": true}, "extra"}, "default": "d", "n": "scalar"}, "status": map[string]any{"status": "Success"}}
	shape := map[string]any{"metadata": map[string]any{"name": "a", "labels": map[string]any{"team": "x"}},
		"spec": map[string]any{"list": []any{map[string]any{"a": 1.0}}, "n": map[string]any{"x": 1.0}}}
	// When projecting onto the configured shape.
	got := project(server, shape)
	// Then only configured keys remain; extra elements and type changes stay visible.
	want := map[string]any{"metadata": map[string]any{"name": "a", "labels": map[string]any{"team": "x"}},
		"spec": map[string]any{"list": []any{map[string]any{"a": 1.0}, "extra"}, "n": "scalar"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestOverlay_whenKeyRemovedFromManifest(t *testing.T) {
	// Given a fresh object, the previously applied manifest and a new one without spec.old.
	fresh := map[string]any{"metadata": map[string]any{"generation": 3.0, "labels": map[string]any{"srv": "1", "team": "a"}}, "spec": map[string]any{"old": "x", "keep": "k", "token": ""}}
	prior := map[string]any{"metadata": map[string]any{"labels": map[string]any{"team": "a"}}, "spec": map[string]any{"old": "x"}}
	want := map[string]any{"metadata": map[string]any{"labels": map[string]any{"team": "b"}}, "spec": map[string]any{"new": "n"}}
	// When overlaying.
	got := overlay(fresh, want, prior)
	// Then managed keys change, dropped managed keys go, server data and blank secrets stay.
	expect := map[string]any{"metadata": map[string]any{"generation": 3.0, "labels": map[string]any{"srv": "1", "team": "b"}}, "spec": map[string]any{"keep": "k", "token": "", "new": "n"}}
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("got %v", got)
	}
}

func TestSecretValues_whenSensitivePathsSet(t *testing.T) {
	m := testManifest(t)
	for _, tc := range []struct{ kind, text, want string }{
		{"ExternalIdentity", "apiVersion: integrations/v3\nkind: ExternalIdentity\nmetadata: {name: mcp}\nspec: {bearerToken: {token: x}}\n",
			"spec.bearerToken.token is a secret. Set it with the wayfinder_external_identity.bearer_token.token_wo attribute, or in manifest_wo, so it is never stored in state."},
		{"Integration", "apiVersion: core/v3\nkind: Integration\nmetadata: {name: i, workspace: w}\nspec: {headers: {a: v}}\n",
			"spec.headers is a secret. Set it with the wayfinder_integration.headers_wo attribute"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			// Given a manifest with a non-empty secret. When scanning it.
			d, err := parseManifest(m, tc.text)
			if err != nil {
				t.Fatal(err)
			}

			s := secretValues(d.kind, d.object)
			// Then one secret is reported with the write-only attribute to use.
			if len(s) != 1 || !strings.HasPrefix(secretMessage(d.kind, s[0]), tc.want) {
				t.Fatalf("secrets = %v", s)
			}
		})
	}
}

func TestSecretValues_whenBlankOrManifestOnly(t *testing.T) {
	m := testManifest(t)
	// Given blank secrets, which keep the stored value server-side.
	d, err := parseManifest(m, "apiVersion: integrations/v3\nkind: ExternalIdentity\nmetadata: {name: mcp}\nspec: {bearerToken: {token: \"\"}}\n")
	if err != nil {
		t.Fatal(err)
	}
	// When scanning. Then nothing is reported.
	if s := secretValues(d.kind, d.object); len(s) != 0 {
		t.Fatalf("secrets = %v", s)
	}
	// And a manifest-only kind's secret message does not point at a nonexistent resource.
	k := findKind(t, m, "Pipeline")
	if msg := secretMessage(k, secretValue{path: "spec.x"}); strings.Contains(msg, "attribute") || !strings.Contains(msg, "never stores secrets in state") {
		t.Fatalf("message = %s", msg)
	}
}
