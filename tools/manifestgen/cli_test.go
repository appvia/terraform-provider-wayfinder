//go:build wfmanifestgen

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/appvia/wayfinder/hack/tfprovider-manifestgen/manifest"
	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
	orgv3 "github.com/appvia/wayfinder/server/org/apis/org/v3"
	"github.com/stretchr/testify/require"
)

func TestRun_writes_valid_manifest_to_requested_output(t *testing.T) {
	// Given real overrides and an isolated output path.
	out := filepath.Join(t.TempDir(), "manifest.json")
	args := []string{"-kinds", os.Getenv("MANIFESTGEN_KINDS"), "-fields", os.Getenv("MANIFESTGEN_FIELDS"),
		"-wayfinder-version", "v3.3.2", "-o", out}
	var stdout bytes.Buffer
	// When the CLI generates a manifest.
	err := run(args, &stdout)
	// Then the requested output is a valid contract document, not stdout.
	require.NoError(t, err)
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	_, err = manifest.Parse(data)
	require.NoError(t, err)
	require.Empty(t, stdout.String())
	require.Equal(t, byte('\n'), data[len(data)-1])
}

func TestRun_fails_without_writing_when_override_path_is_stale(t *testing.T) {
	// Given a valid kind with an invalid fields.yaml path.
	dir := t.TempDir()
	fields := filepath.Join(dir, "fields.yaml")
	out := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(fields, []byte("fields:\n  - {group: org, kind: Workspace, path: spec.doesNotExist, computed: true}\n"), 0o600))
	args := []string{"-kinds", os.Getenv("MANIFESTGEN_KINDS"), "-fields", fields,
		"-wayfinder-version", "v3.3.2", "-o", out}
	// When the CLI attempts generation.
	err := run(args, &bytes.Buffer{})
	// Then entry and tag are reported and no partial output is emitted.
	require.ErrorContains(t, err, "fields[0] org/Workspace spec.doesNotExist at v3.3.2")
	_, statErr := os.Stat(out)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRun_list_kinds_preserves_inventory_mode(t *testing.T) {
	// Given no generation flags.
	var stdout bytes.Buffer
	// When only inventory is requested.
	err := run([]string{"-list-kinds"}, &stdout)
	// Then all registered kinds remain discoverable.
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "org Workspace workspaces tenant")
	require.Contains(t, stdout.String(), "cloudaccess WorkloadIdentity")
}

func TestGenerate_wait_requires_controller_status_and_kind_policy(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		t.Run(map[bool]string{false: "controller", true: "policy"}[noWait], func(t *testing.T) {
			// Given a status-aware kind with a controller and optional noWait policy.
			g := generator{registry: registeredKinds([]corev3.Object{&orgv3.Workspace{}}), version: "v3.3.2", controllers: map[string]bool{"Workspace": true}}
			kinds := KindsFile{Kinds: []KindEntry{{Group: "org", Kind: "Workspace", Class: ClassTyped, NoWait: noWait, Reason: "fixture"}}}
			// When generated.
			m, err := g.generate(kinds, FieldsFile{})
			// Then policy can disable an otherwise eligible wait.
			require.NoError(t, err)
			require.True(t, m.Kinds[0].HasStatus)
			require.Equal(t, !noWait, m.Kinds[0].WaitForReady)
		})
	}
}

func TestGenerate_metadata_only_kind_has_empty_spec(t *testing.T) {
	// Given Label, which carries metadata only and has no Spec field.
	g := generator{registry: registeredKinds([]corev3.Object{&orgv3.Label{}}), version: "v3.3.2"}
	kinds := KindsFile{Kinds: []KindEntry{{Group: "org", Kind: "Label", Class: ClassTyped}}}
	// When generated.
	m, err := g.generate(kinds, FieldsFile{})
	// Then metadata is not accidentally promoted into Spec.
	require.NoError(t, err)
	require.Equal(t, "object", m.Kinds[0].Spec.Type)
	require.Empty(t, m.Kinds[0].Spec.Properties)
}
