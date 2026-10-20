// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"strings"
	"testing"
)

func TestValidate_accepts_group_then_kind_order(t *testing.T) {
	// Given kind names deliberately out of global alphabetic order.
	m := baseManifest()
	m.Kinds = []Kind{m.Kinds[2], m.Kinds[0]}
	m.Kinds[0].Group = "a"
	m.Kinds[1].Group = "b"
	// When validated.
	err := m.Validate()
	// Then group order wins over kind name order.
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidate_rejects_unsorted_groups(t *testing.T) {
	// Given globally sorted names with groups in reverse order.
	m := baseManifest()
	m.Kinds[0].Group = "z"
	// When validated.
	err := m.Validate()
	// Then the group/kind ordering contract is enforced.
	if err == nil || !strings.Contains(err.Error(), "sorted by group, kind") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_accepts_APIPath_resource_segment(t *testing.T) {
	// Given the relative resource path returned by Wayfinder APIPath().
	m := baseManifest()
	m.Kinds[0].APIPath = "alphas"
	// When validated.
	err := m.Validate()
	// Then native API path metadata is preserved.
	if err != nil {
		t.Fatal(err)
	}
}
