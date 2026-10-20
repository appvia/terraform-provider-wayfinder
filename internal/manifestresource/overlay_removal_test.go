// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifestresource

import (
	"reflect"
	"testing"
)

func TestOverlay_whenOnlyManagedKeyRemoved(t *testing.T) {
	for _, field := range []string{"labels", "spec"} {
		t.Run(field, func(t *testing.T) {
			// Given: a managed key alongside a never-managed server key.
			fresh := map[string]any{field: map[string]any{"a": "1", "b": "2", "server": "kept"}}
			prior := map[string]any{field: map[string]any{"a": "1", "b": "2"}}
			want := map[string]any{field: map[string]any{"a": "1"}}

			// When: only key b is removed from the applied manifest.
			got := overlay(fresh, want, prior)

			// Then: b is removed while unmanaged data and the fresh input survive.
			expect := map[string]any{field: map[string]any{"a": "1", "server": "kept"}}
			if !reflect.DeepEqual(got, expect) {
				t.Fatalf("overlay = %#v, want %#v", got, expect)
			}
			if !reflect.DeepEqual(fresh, map[string]any{field: map[string]any{"a": "1", "b": "2", "server": "kept"}}) {
				t.Fatalf("fresh input mutated: %#v", fresh)
			}
		})
	}
}
