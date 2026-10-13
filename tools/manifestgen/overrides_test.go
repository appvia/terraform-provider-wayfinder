//go:build wfmanifestgen

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKinds_accepts_valid_file(t *testing.T) {
	// Given a sorted file with each class
	data := []byte(`kinds:
  - {group: a, kind: A, class: typed, name: wayfinder_a_thing}
  - {group: a, kind: B, class: manifest, noWait: true, reason: never reconciles}
  - {group: b, kind: A, class: exclude, reason: internal}
`)
	// When parsed
	kf, err := ParseKinds(data)
	// Then every entry is decoded
	require.NoError(t, err)
	assert.Equal(t, []KindEntry{
		{Group: "a", Kind: "A", Class: ClassTyped, Name: "wayfinder_a_thing"},
		{Group: "a", Kind: "B", Class: ClassManifest, NoWait: true, Reason: "never reconciles"},
		{Group: "b", Kind: "A", Class: ClassExclude, Reason: "internal"},
	}, kf.Kinds)
}

func TestParseKinds_rejects_invalid_files(t *testing.T) {
	cases := map[string]struct {
		yaml string
		want string
	}{
		"unknown key": {
			yaml: "kinds:\n  - {group: a, kind: A, class: typed, colour: red}\n",
			want: `unknown field "colour"`,
		},
		"unknown top-level key": {
			yaml: "kinds: []\nextra: 1\n",
			want: `unknown field "extra"`,
		},
		"exclude without reason": {
			yaml: "kinds:\n  - {group: a, kind: A, class: exclude}\n",
			want: "a/A: class exclude requires a reason",
		},
		"noWait without reason": {
			yaml: "kinds:\n  - {group: a, kind: A, class: typed, noWait: true}\n",
			want: "a/A: noWait requires a reason",
		},
		"invalid class": {
			yaml: "kinds:\n  - {group: a, kind: A, class: generic}\n",
			want: `a/A: invalid class "generic"`,
		},
		"missing kind": {
			yaml: "kinds:\n  - {group: a, class: typed}\n",
			want: "group and kind are required",
		},
		"duplicate": {
			yaml: "kinds:\n  - {group: a, kind: A, class: typed}\n  - {group: a, kind: A, class: typed}\n",
			want: "kinds[1] a/A: duplicate entry",
		},
		"unsorted by kind": {
			yaml: "kinds:\n  - {group: a, kind: B, class: typed}\n  - {group: a, kind: A, class: typed}\n",
			want: "kinds[1] a/A: entries must be sorted",
		},
		"unsorted by group": {
			yaml: "kinds:\n  - {group: b, kind: A, class: typed}\n  - {group: a, kind: Z, class: typed}\n",
			want: "kinds[1] a/Z: entries must be sorted",
		},
		"name on exclude": {
			yaml: "kinds:\n  - {group: a, kind: A, class: exclude, reason: x, name: wayfinder_a}\n",
			want: "meaningless on an excluded kind",
		},
		"bad name": {
			yaml: "kinds:\n  - {group: a, kind: A, class: typed, name: a-thing}\n",
			want: `name "a-thing" must match`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseKinds([]byte(tc.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestParseFields_accepts_valid_file(t *testing.T) {
	data := []byte(`fields:
  - {group: a, kind: A, path: spec.x, immutable: true, computed: true}
  - {group: a, kind: A, path: spec.y.z, ignore: true, reason: server owned}
`)
	ff, err := ParseFields(data)
	require.NoError(t, err)
	assert.Len(t, ff.Fields, 2)
}

func TestParseFields_accepts_empty_list(t *testing.T) {
	ff, err := ParseFields([]byte("fields: []\n"))
	require.NoError(t, err)
	assert.Empty(t, ff.Fields)
}

func TestParseFields_rejects_invalid_files(t *testing.T) {
	cases := map[string]struct {
		yaml string
		want string
	}{
		"unknown key": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.x, immutable: true, sensitive: true}\n",
			want: `unknown field "sensitive"`,
		},
		"ignore with another flag": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.x, ignore: true, computed: true, reason: r}\n",
			want: "ignore cannot be combined",
		},
		"ignore without reason": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.x, ignore: true}\n",
			want: "ignore requires a reason",
		},
		"no flag": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.x}\n",
			want: "set at least one of",
		},
		"path outside spec": {
			yaml: "fields:\n  - {group: a, kind: A, path: status.x, computed: true}\n",
			want: "path must be a dotted path starting with spec.",
		},
		"duplicate": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.x, computed: true}\n  - {group: a, kind: A, path: spec.x, immutable: true}\n",
			want: "duplicate entry",
		},
		"unsorted by path": {
			yaml: "fields:\n  - {group: a, kind: A, path: spec.y, computed: true}\n  - {group: a, kind: A, path: spec.x, computed: true}\n",
			want: "fields[1] a/A spec.x: entries must be sorted",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseFields([]byte(tc.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestCheckFieldKinds_rejects_excluded_and_unknown_kinds(t *testing.T) {
	kinds := KindsFile{Kinds: []KindEntry{
		{Group: "a", Kind: "A", Class: ClassTyped},
		{Group: "a", Kind: "B", Class: ClassExclude, Reason: "r"},
	}}
	fields := FieldsFile{Fields: []FieldEntry{
		{Group: "a", Kind: "A", Path: "spec.x", Computed: true},
		{Group: "a", Kind: "B", Path: "spec.x", Computed: true},
		{Group: "a", Kind: "C", Path: "spec.x", Computed: true},
	}}
	err := CheckFieldKinds(kinds, fields)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a/B spec.x: kind is excluded")
	assert.Contains(t, err.Error(), "a/C spec.x: kind is not listed")
	assert.NotContains(t, err.Error(), "a/A")
}
