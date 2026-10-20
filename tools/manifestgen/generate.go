// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build wfmanifestgen

package main

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/appvia/wayfinder/hack/tfprovider-manifestgen/manifest"
	corev3 "github.com/appvia/wayfinder/pkg/apis/core/v3"
)

type generator struct {
	registry    []RegisteredKind
	controllers map[string]bool
	version     string
	commit      string
}

func (g generator) generate(kinds KindsFile, fields FieldsFile) (*manifest.Manifest, error) {
	if err := validateKinds(kinds); err != nil {
		return nil, err
	}
	if err := validateFields(fields); err != nil {
		return nil, err
	}
	if err := CheckCoverage(kinds, g.registry, g.version); err != nil {
		return nil, err
	}
	if err := CheckFieldKinds(kinds, fields); err != nil {
		return nil, err
	}
	objects := make(map[GroupKind]RegisteredKind, len(g.registry))
	externalIdentity, encrypted := false, false
	seen := make(map[reflect.Type]bool)
	for _, rk := range g.registry {
		objects[rk.GroupKind] = rk
		externalIdentity = externalIdentity || rk.Kind == "ExternalIdentity"
		encrypted = hasEncryptionMetadata(reflect.TypeOf(rk.Object), seen) || encrypted
	}
	if externalIdentity && !encrypted {
		return nil, fmt.Errorf("%s: pinned Wayfinder tag lacks EncryptedFields metadata; pin a newer tag", g.version)
	}
	m := &manifest.Manifest{ManifestVersion: manifest.ManifestVersion, WayfinderVersion: g.version,
		WayfinderCommit: g.commit, Kinds: []manifest.Kind{}}
	for _, entry := range kinds.VarScopes {
		if entry.Class == ClassTyped {
			m.VarScopes = append(m.VarScopes, entry.Scope)
		}
	}
	slices.Sort(m.VarScopes)
	for _, entry := range kinds.Kinds {
		switch entry.Class {
		case ClassExclude:
			m.Excluded = append(m.Excluded, entry.Kind)
		case ClassTyped, ClassManifest:
			kind, err := g.kind(objects[entry.GroupKind()], entry)
			if err != nil {
				return nil, fmt.Errorf("%s at %s: %w", entry.GroupKind(), g.version, err)
			}
			m.Kinds = append(m.Kinds, kind)
		default:
			return nil, fmt.Errorf("%s: unknown class %q", entry.GroupKind(), entry.Class)
		}
	}
	slices.Sort(m.Excluded)
	if err := applyOverrides(m, fields, g.version); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("generated manifest at %s: %w", g.version, err)
	}
	return m, nil
}

func (g generator) kind(rk RegisteredKind, entry KindEntry) (manifest.Kind, error) {
	obj := rk.Object
	gvk := obj.GroupVersionKind()
	_, status := obj.(corev3.CommonStatusAware)
	k := manifest.Kind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind,
		APIPath: obj.APIPath(), Typed: entry.Class == ClassTyped, Versioned: corev3.IsVersioned(obj),
		HasStatus: status, WaitForReady: status && g.controllers[gvk.Kind] && !entry.NoWait}
	for _, scope := range []struct {
		present bool
		name    string
	}{
		{obj.SupportedScopes().HasTenant(), "tenant"},
		{obj.SupportedScopes().HasWorkspace(), "workspace"},
		{obj.SupportedScopes().HasEnvironment(), "environment"},
	} {
		if scope.present {
			k.Scopes = append(k.Scopes, scope.name)
		}
	}
	if k.Typed {
		k.TerraformName = entry.Name
		if k.TerraformName == "" {
			k.TerraformName = "wayfinder_" + snake(gvk.Kind)
		}
	}
	t := indirect(reflect.TypeOf(obj))
	k.Description = metadataFor(t).docs[""]
	spec, ok := t.FieldByName("Spec")
	if !ok {
		k.Spec = &manifest.Field{Type: "object"}
		return k, nil
	}
	field, err := reflectField(spec.Type, make(map[reflect.Type]bool))
	if err != nil {
		return k, fmt.Errorf("reflect spec: %w", err)
	}
	if field.Type == "json" && indirect(spec.Type).Kind() == reflect.Struct && indirect(spec.Type).NumField() == 0 {
		field.Type = "object"
	}
	k.Spec = field
	return k, nil
}

func applyOverrides(m *manifest.Manifest, fields FieldsFile, tag string) error {
	byKind := make(map[GroupKind]*manifest.Field, len(m.Kinds))
	for _, k := range m.Kinds {
		byKind[GroupKind{Group: k.Group, Kind: k.Kind}] = k.Spec
	}
	for i, entry := range fields.Fields {
		parent := byKind[entry.GroupKind()]
		parts := strings.Split(entry.Path, ".")[1:]
		var field *manifest.Field
		for j, part := range parts {
			field = nil
			for parent != nil && (parent.Type == "array" || parent.Type == "map") {
				switch parent.Type {
				case "array":
					parent = parent.Items
				case "map":
					parent = parent.Values
				}
			}
			if parent == nil {
				break
			}
			field = parent.Properties[part]
			if field == nil || j == len(parts)-1 {
				break
			}
			parent = field
		}
		if field == nil {
			return fmt.Errorf("fields[%d] %s %s at %s: path does not resolve", i, entry.GroupKind(), entry.Path, tag)
		}
		if entry.Ignore {
			name := parts[len(parts)-1]
			delete(parent.Properties, name)
			parent.PropertyOrder = slices.DeleteFunc(parent.PropertyOrder, func(s string) bool { return s == name })
		} else {
			field.Immutable, field.Computed = entry.Immutable, entry.Computed
		}
	}
	return nil
}
