// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRewordAppliesRules(t *testing.T) {
	got, err := reword("Defaults to this instance's canonical Wayfinder audience for the tenant.")
	if err != nil {
		t.Fatal(err)
	}

	if want := "Defaults to the canonical Wayfinder audience for the tenant."; got != want {
		t.Fatalf("reword = %q, want %q", got, want)
	}
}

func TestRewordRejectsForbiddenWording(t *testing.T) {
	for _, text := range []string{
		"Ask your administrator.",
		"Stored in a VarSet.",
		"Set WF_FOO on the server.",
		"Set the Wayfinder Helm values.",
		"Configured on this instance.",
	} {
		if _, err := reword(text); err == nil {
			t.Errorf("reword(%q) succeeded, want error", text)
		}
	}
}

func TestRewordAllowsTenantHelmAndAttributeNames(t *testing.T) {
	for _, text := range []string{"Helm values for your chart.", "WFAgent installs the agent (`wf_agent`)."} {
		if got, err := reword(text); err != nil || got != text {
			t.Errorf("reword(%q) = %q, %v", text, got, err)
		}
	}
}

func TestRemoveStaleKeepsHandWrittenExamples(t *testing.T) {
	dir := t.TempDir()
	o := options{out: filepath.Join(dir, "gen"), examples: filepath.Join(dir, "examples")}
	hand := filepath.Join(o.examples, "resources", "wayfinder_variable", "resource.tf")

	stale := filepath.Join(o.examples, "resources", "wayfinder_gone", "resource.tf")
	quoting := filepath.Join(o.out, "fixture_test.go")

	for path, body := range map[string]string{
		hand:    "resource \"wayfinder_variable\" \"x\" {}\n",
		stale:   header + "x\n",
		quoting: "package generated\n\nconst fixture = `\n// " + generatedMarker + "\n`\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := removeStale(map[string][]byte{}, o); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(hand); err != nil {
		t.Fatalf("hand-written example removed: %v", err)
	}

	if _, err := os.Stat(quoting); err != nil {
		t.Fatalf("hand-written file quoting the marker removed: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale generated example kept: %v", err)
	}
}
