//go:build wfmanifestgen

package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"
)

// KindClass is the Terraform treatment of an API kind (D-OVR).
type KindClass string

const (
	// ClassTyped kinds get a generated, fully typed resource + data source.
	ClassTyped KindClass = "typed"
	// ClassManifest kinds are only manageable through wayfinder_manifest.
	ClassManifest KindClass = "manifest"
	// ClassExclude kinds are not exposed by the provider at all.
	ClassExclude KindClass = "exclude"
)

// KindEntry is one entry of kinds.yaml.
type KindEntry struct {
	Group  string    `json:"group"`
	Kind   string    `json:"kind"`
	Class  KindClass `json:"class"`
	Reason string    `json:"reason,omitempty"`
	Name   string    `json:"name,omitempty"`
	NoWait bool      `json:"noWait,omitempty"`
}

// GroupKind returns the entry's identity.
func (k KindEntry) GroupKind() GroupKind { return GroupKind{Group: k.Group, Kind: k.Kind} }

// VarScopeEntry classifies one scope served by the vars API.
type VarScopeEntry struct {
	Scope  string    `json:"scope"`
	Class  KindClass `json:"class"`
	Reason string    `json:"reason,omitempty"`
}

// KindsFile is the top-level shape of kinds.yaml.
type KindsFile struct {
	Kinds     []KindEntry     `json:"kinds"`
	VarScopes []VarScopeEntry `json:"varScopes"`
}

// FieldEntry is one entry of fields.yaml.
type FieldEntry struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Immutable bool   `json:"immutable,omitempty"`
	Computed  bool   `json:"computed,omitempty"`
	Ignore    bool   `json:"ignore,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// GroupKind returns the identity of the kind the entry applies to.
func (f FieldEntry) GroupKind() GroupKind { return GroupKind{Group: f.Group, Kind: f.Kind} }

// FieldsFile is the top-level shape of fields.yaml.
type FieldsFile struct {
	Fields []FieldEntry `json:"fields"`
}

var terraformNameRE = regexp.MustCompile(`^wayfinder_[a-z][a-z0-9_]*$`)

// LoadKinds reads and validates kinds.yaml.
func LoadKinds(path string) (KindsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KindsFile{}, err
	}
	kf, err := ParseKinds(data)
	if err != nil {
		return KindsFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return kf, nil
}

// ParseKinds strictly decodes and validates kinds.yaml content.
func ParseKinds(data []byte) (KindsFile, error) {
	var kf KindsFile
	if err := yaml.UnmarshalStrict(data, &kf); err != nil {
		return KindsFile{}, fmt.Errorf("decoding kinds: %w", err)
	}
	return kf, validateKinds(kf)
}

func validateKinds(kf KindsFile) error {
	var errs []error
	for i, k := range kf.Kinds {
		at := fmt.Sprintf("kinds[%d] %s", i, k.GroupKind())
		if k.Group == "" || k.Kind == "" {
			errs = append(errs, fmt.Errorf("%s: group and kind are required", at))
		}
		switch k.Class {
		case ClassTyped, ClassManifest:
			if k.Name != "" && !terraformNameRE.MatchString(k.Name) {
				errs = append(errs, fmt.Errorf("%s: name %q must match %s", at, k.Name, terraformNameRE))
			}
		case ClassExclude:
			if strings.TrimSpace(k.Reason) == "" {
				errs = append(errs, fmt.Errorf("%s: class exclude requires a reason", at))
			}
			if k.Name != "" || k.NoWait {
				errs = append(errs, fmt.Errorf("%s: name/noWait are meaningless on an excluded kind", at))
			}
		default:
			errs = append(errs, fmt.Errorf("%s: invalid class %q (want typed|manifest|exclude)", at, k.Class))
		}
		if k.NoWait && strings.TrimSpace(k.Reason) == "" {
			errs = append(errs, fmt.Errorf("%s: noWait requires a reason", at))
		}
		if i > 0 {
			errs = append(errs, checkOrder(at, compareGroupKind(kf.Kinds[i-1].GroupKind(), k.GroupKind())))
		}
	}
	for i, scope := range kf.VarScopes {
		at := fmt.Sprintf("varScopes[%d] %s", i, scope.Scope)
		if scope.Scope == "" {
			errs = append(errs, fmt.Errorf("%s: scope is required", at))
		}
		switch scope.Class {
		case ClassTyped:
		case ClassExclude:
			if strings.TrimSpace(scope.Reason) == "" {
				errs = append(errs, fmt.Errorf("%s: class exclude requires a reason", at))
			}
		default:
			errs = append(errs, fmt.Errorf("%s: invalid class %q (want typed|exclude)", at, scope.Class))
		}
		if i > 0 {
			errs = append(errs, checkOrder(at, cmp.Compare(kf.VarScopes[i-1].Scope, scope.Scope)))
		}
	}
	return errors.Join(errs...)
}

// LoadFields reads and validates fields.yaml.
func LoadFields(path string) (FieldsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FieldsFile{}, err
	}
	ff, err := ParseFields(data)
	if err != nil {
		return FieldsFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return ff, nil
}

// ParseFields strictly decodes and validates fields.yaml content.
func ParseFields(data []byte) (FieldsFile, error) {
	var ff FieldsFile
	if err := yaml.UnmarshalStrict(data, &ff); err != nil {
		return FieldsFile{}, fmt.Errorf("decoding fields: %w", err)
	}
	return ff, validateFields(ff)
}

func compareField(a, b FieldEntry) int {
	return cmp.Or(compareGroupKind(a.GroupKind(), b.GroupKind()), cmp.Compare(a.Path, b.Path))
}

func validateFields(ff FieldsFile) error {
	var errs []error
	for i, f := range ff.Fields {
		at := fmt.Sprintf("fields[%d] %s %s", i, f.GroupKind(), f.Path)
		if f.Group == "" || f.Kind == "" {
			errs = append(errs, fmt.Errorf("%s: group and kind are required", at))
		}
		if !strings.HasPrefix(f.Path, "spec.") || strings.HasSuffix(f.Path, ".") || strings.Contains(f.Path, "..") {
			errs = append(errs, fmt.Errorf("%s: path must be a dotted path starting with spec.", at))
		}
		switch {
		case f.Ignore && (f.Immutable || f.Computed):
			errs = append(errs, fmt.Errorf("%s: ignore cannot be combined with immutable or computed", at))
		case f.Ignore && strings.TrimSpace(f.Reason) == "":
			errs = append(errs, fmt.Errorf("%s: ignore requires a reason", at))
		case !f.Ignore && !f.Immutable && !f.Computed:
			errs = append(errs, fmt.Errorf("%s: set at least one of immutable, computed, ignore", at))
		}
		if i > 0 {
			errs = append(errs, checkOrder(at, compareField(ff.Fields[i-1], f)))
		}
	}
	return errors.Join(errs...)
}

// checkOrder turns the comparison of an entry with its predecessor into an
// error for duplicates (0) or out-of-order entries (>0).
func checkOrder(at string, c int) error {
	switch {
	case c == 0:
		return fmt.Errorf("%s: duplicate entry", at)
	case c > 0:
		return fmt.Errorf("%s: entries must be sorted by group, kind (, path), or scope for varScopes", at)
	default:
		return nil
	}
}

// CheckFieldKinds fails for fields.yaml entries whose kind is not typed or
// manifest in kinds.yaml. Path existence is checked during generation.
func CheckFieldKinds(kinds KindsFile, fields FieldsFile) error {
	class := make(map[GroupKind]KindClass, len(kinds.Kinds))
	for _, k := range kinds.Kinds {
		class[k.GroupKind()] = k.Class
	}
	var errs []error
	for _, f := range fields.Fields {
		switch c := class[f.GroupKind()]; c {
		case ClassTyped, ClassManifest:
		case ClassExclude:
			errs = append(errs, fmt.Errorf("fields.yaml %s %s: kind is excluded", f.GroupKind(), f.Path))
		default:
			errs = append(errs, fmt.Errorf("fields.yaml %s %s: kind is not listed in kinds.yaml", f.GroupKind(), f.Path))
		}
	}
	return errors.Join(errs...)
}
