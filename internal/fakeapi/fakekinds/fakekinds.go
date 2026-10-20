// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

// Package fakekinds registers manifest kinds with the fake Wayfinder API, so provider-level
// tests can exercise the generated resources against exactly the kinds compiled into the
// provider.
//
// It derives each fakeapi.KindSpec from a manifest.Kind: identity (group, version, apiPath),
// scopes, versioning, status, and the sensitive (+encrypted) leaves, which the fake blanks on
// GET and backfills on PUT (plan R9). Paths follow the fakeapi convention: dotted JSON paths
// from the object root, array items addressed by the array field name.
package fakekinds

import (
	"slices"

	"github.com/appvia/terraform-provider-wayfinder/internal/fakeapi"
	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

// Spec returns the fake registration for one manifest kind.
func Spec(k manifest.Kind) fakeapi.KindSpec {
	s := fakeapi.KindSpec{Kind: k.Kind, Group: k.Group, Version: k.Version, APIPath: k.APIPath, Versioned: k.Versioned, HasStatus: k.HasStatus}
	for _, scope := range k.Scopes {
		s.Scopes = append(s.Scopes, fakeapi.Scope(scope))
	}

	if k.Spec != nil {
		collectSensitive(&s, "spec", k.Spec)
	}

	return s
}

// RegisterAll registers every kind of m (typed and manifest-only) with s.
func RegisterAll(s *fakeapi.Server, m *manifest.Manifest) {
	for _, k := range m.Kinds {
		s.RegisterKind(Spec(k))
	}
}

func collectSensitive(s *fakeapi.KindSpec, path string, f *manifest.Field) {
	if f.Sensitive {
		if f.Type == "map" {
			s.SensitiveMaps = append(s.SensitiveMaps, path)
		} else {
			s.SensitiveStrings = append(s.SensitiveStrings, path)
		}

		return
	}

	if f.Items != nil {
		collectSensitive(s, path, f.Items)
	}

	for _, name := range orderedNames(f) {
		collectSensitive(s, path+"."+name, f.Properties[name])
	}
}

// orderedNames returns property names in document order, then any unlisted names sorted, so the
// registration is deterministic.
func orderedNames(f *manifest.Field) []string {
	seen := make(map[string]bool, len(f.Properties))

	names := make([]string, 0, len(f.Properties))
	for _, n := range f.PropertyOrder {
		if _, ok := f.Properties[n]; ok && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}

	var rest []string

	for n := range f.Properties {
		if !seen[n] {
			rest = append(rest, n)
		}
	}

	slices.Sort(rest)

	return append(names, rest...)
}
