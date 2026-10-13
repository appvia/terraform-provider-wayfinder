//go:build wfmanifestgen

// Command tfprovider-manifestgen derives the Terraform provider manifest from
// the Wayfinder API types. It is compiled only inside a checkout of the pinned
// Wayfinder tag by scripts/manifestgen.sh (see DEVELOPMENT.md).
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
)

type options struct {
	kindsPath        string
	fieldsPath       string
	wayfinderVersion string
	wayfinderCommit  string
	outPath          string
	listKinds        bool
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "manifestgen:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	var o options
	fs := flag.NewFlagSet("manifestgen", flag.ContinueOnError)
	fs.StringVar(&o.kindsPath, "kinds", "", "path to kinds.yaml")
	fs.StringVar(&o.fieldsPath, "fields", "", "path to fields.yaml")
	fs.StringVar(&o.wayfinderVersion, "wayfinder-version", "", "pinned Wayfinder release tag")
	fs.StringVar(&o.wayfinderCommit, "wayfinder-commit", "", "commit SHA of the pinned tag")
	fs.StringVar(&o.outPath, "o", "", "output manifest path")
	fs.BoolVar(&o.listKinds, "list-kinds", false, "print every registered kind and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	registry := wayfinderRegistry()
	if o.listKinds {
		return listKinds(stdout, registry)
	}
	if o.kindsPath == "" || o.fieldsPath == "" || o.wayfinderVersion == "" {
		return errors.New("-kinds, -fields and -wayfinder-version are required")
	}

	kinds, err := LoadKinds(o.kindsPath)
	if err != nil {
		return err
	}
	fields, err := LoadFields(o.fieldsPath)
	if err != nil {
		return err
	}
	if err := CheckCoverage(kinds, registry, o.wayfinderVersion); err != nil {
		return err
	}
	if err := CheckFieldKinds(kinds, fields); err != nil {
		return err
	}

	// Manifest generation lands in a later change; overrides and coverage are
	// already enforced above.
	_, err = fmt.Fprintln(stdout, "manifestgen: overrides valid; manifest generation not implemented yet")
	return err
}

func listKinds(w io.Writer, registry []RegisteredKind) error {
	for _, rk := range registry {
		path := rk.APIPath
		if path == "" {
			path = "-"
		}
		_, err := fmt.Fprintf(w, "%s %s %s %s alpha=%t hidden=%t readOnly=%t\n",
			rk.Group, rk.Kind, path, scopeList(rk.Scopes), rk.Alpha, rk.Hidden, rk.ReadOnly)
		if err != nil {
			return err
		}
	}
	return nil
}

// scopeList renders a scope bitmask as e.g. "tenant|workspace".
func scopeList(s corev3.ObjectScope) string {
	var parts []string
	for _, p := range []struct {
		has  bool
		name string
	}{
		{s.HasSystem(), "system"},
		{s.HasTenant(), "tenant"},
		{s.HasWorkspace(), "workspace"},
		{s.HasEnvironment(), "environment"},
	} {
		if p.has {
			parts = append(parts, p.name)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "|")
}
