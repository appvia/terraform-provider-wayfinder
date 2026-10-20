// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package convert_test

import (
	"reflect"
	"testing"

	"github.com/appvia/terraform-provider-wayfinder/internal/convert"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func TestParseImportID_whenScopeAndVersionVary(t *testing.T) {
	for _, tc := range []struct {
		id              string
		scopes          []string
		versioned, fail bool
		want            convert.Identity
	}{
		{"name", []string{"tenant"}, false, false, convert.Identity{Name: "name"}},
		{"ws/name", []string{"workspace"}, false, false, convert.Identity{Workspace: "ws", Name: "name"}},
		{"ws/env/name", []string{"environment"}, false, false, convert.Identity{Workspace: "ws", Environment: "env", Name: "name"}},
		{"ws/name@1.2.3", []string{"workspace"}, true, false, convert.Identity{Workspace: "ws", Name: "name", Version: "1.2.3"}},
		{"ws/env/name", []string{"workspace", "environment"}, false, false, convert.Identity{Workspace: "ws", Environment: "env", Name: "name"}},
		{"a/b/c/d", []string{"environment"}, false, true, convert.Identity{}},
		{"name", []string{"workspace"}, false, true, convert.Identity{}},
		{"ws/name", []string{"tenant"}, false, true, convert.Identity{}},
		{"ws//name", []string{"environment"}, false, true, convert.Identity{}},
		{"name@1.2.3", []string{"tenant"}, false, true, convert.Identity{}},
		{"name", []string{"tenant"}, true, true, convert.Identity{}},
		{"name@v1.2.3", []string{"tenant"}, true, true, convert.Identity{}},
		{"name@1.2.3@4.5.6", []string{"tenant"}, true, true, convert.Identity{}},
	} {
		t.Run(tc.id, func(t *testing.T) {
			// Given
			k := manifest.Kind{Scopes: tc.scopes, Versioned: tc.versioned}
			// When
			got, err := convert.ParseImportID(k, tc.id)
			// Then
			if tc.fail {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if got != tc.want || got.ID() != tc.id {
				t.Fatalf("identity: %#v", got)
			}
		})
	}
}

func TestStatusHelpers_whenStatusPresentOrAbsent(t *testing.T) {
	for _, text := range []string{`{"status":{"status":"Failure","message":"failed","detail":"why","conditions":[true]}}`, `{}`, `{"status":null}`} {
		t.Run(text, func(t *testing.T) {
			// Given
			x := object(t, text)
			// When
			s, m, d := convert.StatusSummary(x)
			j, ok := convert.StatusJSON(x)
			// Then
			if x["status"] == nil {
				if ok || s != "" || m != "" || d != "" {
					t.Fatal("unexpected status")
				}

				return
			}

			if s != "Failure" || m != "failed" || d != "why" || !ok {
				t.Fatal("summary missing")
			}

			if !reflect.DeepEqual(object(t, j), x["status"]) {
				t.Fatal("status JSON incomplete")
			}
		})
	}
}
