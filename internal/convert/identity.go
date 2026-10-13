package convert

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

var ErrImportID = errors.New("invalid import ID")

var semverRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-([0-9A-Za-z-]+\.)*[0-9A-Za-z-]+)?(\+([0-9A-Za-z-]+\.)*[0-9A-Za-z-]+)?$`)

// Identity contains plain fields compatible with a client resource reference.
type Identity struct{ Workspace, Environment, Name, Version string }

// ID builds [workspace/[environment/]]name[@version].
func (i Identity) ID() string {
	parts := []string{}
	if i.Workspace != "" {
		parts = append(parts, i.Workspace)
	}
	if i.Environment != "" {
		parts = append(parts, i.Environment)
	}
	parts = append(parts, i.Name)
	id := strings.Join(parts, "/")
	if i.Version != "" {
		id += "@" + i.Version
	}
	return id
}

// ParseImportID checks the selected scope and strict, unprefixed semver version.
func ParseImportID(kind manifest.Kind, id string) (Identity, error) {
	name, version, hasVersion := strings.Cut(id, "@")
	if kind.Versioned != hasVersion {
		return Identity{}, fmt.Errorf("%s: versioned=%t; expected [workspace/[environment/]]name%s: %w", kind.Kind, kind.Versioned, versionSuffix(kind.Versioned), ErrImportID)
	}
	if hasVersion && !validVersion(version) {
		return Identity{}, fmt.Errorf("version must be strict semver without v prefix: %w", ErrImportID)
	}
	parts := strings.Split(name, "/")
	if len(parts) > 3 {
		return Identity{}, fmt.Errorf("expected at most two '/' separators: %w", ErrImportID)
	}
	for _, part := range parts {
		if part == "" || strings.TrimSpace(part) != part {
			return Identity{}, fmt.Errorf("scope and name segments must be nonempty without surrounding whitespace: %w", ErrImportID)
		}
	}
	scope := []string{"tenant", "workspace", "environment"}[len(parts)-1]
	if !slices.Contains(kind.Scopes, scope) {
		return Identity{}, fmt.Errorf("ID selects %s scope, kind supports %s: %w", scope, strings.Join(kind.Scopes, ", "), ErrImportID)
	}
	ident := Identity{Name: parts[len(parts)-1], Version: version}
	if len(parts) >= 2 {
		ident.Workspace = parts[0]
	}
	if len(parts) == 3 {
		ident.Environment = parts[1]
	}
	return ident, nil
}

func versionSuffix(versioned bool) string {
	if versioned {
		return "@version"
	}
	return ""
}

func validVersion(version string) bool {
	if !semverRE.MatchString(version) {
		return false
	}
	core, _, _ := strings.Cut(version, "+")
	_, pre, hasPre := strings.Cut(core, "-")
	if hasPre {
		for _, part := range strings.Split(pre, ".") {
			if len(part) > 1 && part[0] == '0' && strings.Trim(part, "0123456789") == "" {
				return false
			}
		}
	}
	return true
}
