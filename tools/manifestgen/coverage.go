// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build wfmanifestgen

package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
	"github.com/appvia/wayfinder/pkg/schema"
)

// GroupKind identifies an API kind independent of its version.
type GroupKind struct {
	Group string
	Kind  string
}

func (gk GroupKind) String() string { return gk.Group + "/" + gk.Kind }

func compareGroupKind(a, b GroupKind) int {
	return cmp.Or(cmp.Compare(a.Group, b.Group), cmp.Compare(a.Kind, b.Kind))
}

// RegisteredKind is the classification-relevant view of one kind served by
// the Wayfinder registry at the pinned tag.
type RegisteredKind struct {
	GroupKind
	Version    string
	APIPath    string
	Scopes     corev3.ObjectScope
	Alpha      bool
	Hidden     bool
	HideReason string
	ReadOnly   bool
	Object     corev3.Object
}

// registeredKinds converts registry objects (preferred versions) into a
// sorted RegisteredKind list.
func registeredKinds(objs []corev3.Object) []RegisteredKind {
	out := make([]RegisteredKind, 0, len(objs))
	for _, obj := range objs {
		gvk := obj.GroupVersionKind()
		rk := RegisteredKind{
			GroupKind: GroupKind{Group: gvk.Group, Kind: gvk.Kind},
			Version:   gvk.Version,
			APIPath:   obj.APIPath(),
			Scopes:    obj.SupportedScopes(),
			Object:    obj,
		}
		if p, ok := obj.(corev3.APIDocMetaProvider); ok {
			meta := p.APIDocMeta()
			rk.Alpha, rk.Hidden, rk.HideReason, rk.ReadOnly = meta.Alpha, meta.Hidden, meta.HideReason, meta.ReadOnly
		}
		out = append(out, rk)
	}
	slices.SortFunc(out, func(a, b RegisteredKind) int { return compareGroupKind(a.GroupKind, b.GroupKind) })
	return out
}

// wayfinderRegistry returns every kind registered at the pinned tag.
func wayfinderRegistry() []RegisteredKind {
	return registeredKinds(schema.GetAllWayfinderTypesPreferredVersions())
}

// CoverageError lists the differences between kinds.yaml and the registry.
type CoverageError struct {
	Tag     string
	Missing []GroupKind // registered but absent from kinds.yaml
	Stale   []GroupKind // listed in kinds.yaml but not registered
}

func (e *CoverageError) Error() string {
	var b strings.Builder
	b.WriteString("kinds.yaml does not match the Wayfinder registry:")
	for _, gk := range e.Missing {
		fmt.Fprintf(&b, "\n  - %s is registered at %s but missing from kinds.yaml", gk, e.Tag)
	}
	for _, gk := range e.Stale {
		fmt.Fprintf(&b, "\n  - %s is listed in kinds.yaml but no longer exists at %s", gk, e.Tag)
	}
	return b.String()
}

// CheckCoverage fails unless kinds.yaml lists exactly the registered kinds.
func CheckCoverage(kinds KindsFile, registry []RegisteredKind, tag string) error {
	listed := make(map[GroupKind]bool, len(kinds.Kinds))
	for _, k := range kinds.Kinds {
		listed[k.GroupKind()] = true
	}
	registered := make(map[GroupKind]bool, len(registry))
	cerr := &CoverageError{Tag: tag}
	for _, rk := range registry {
		registered[rk.GroupKind] = true
		if !listed[rk.GroupKind] {
			cerr.Missing = append(cerr.Missing, rk.GroupKind)
		}
	}
	for _, k := range kinds.Kinds {
		if !registered[k.GroupKind()] {
			cerr.Stale = append(cerr.Stale, k.GroupKind())
		}
	}
	if len(cerr.Missing) == 0 && len(cerr.Stale) == 0 {
		return nil
	}
	slices.SortFunc(cerr.Missing, compareGroupKind)
	slices.SortFunc(cerr.Stale, compareGroupKind)
	return cerr
}
