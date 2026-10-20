// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"fmt"
	"os"

	"github.com/appvia/terraform-provider-wayfinder/schema"
)

// Load reads, decodes, and validates a manifest file at path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}

	m, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	return m, nil
}

// LoadEmbedded decodes and validates the manifest compiled into the provider.
func LoadEmbedded() (*Manifest, error) {
	m, err := Parse(schema.Manifest)
	if err != nil {
		return nil, fmt.Errorf("parse embedded manifest: %w", err)
	}

	return m, nil
}

// FindKind returns the kind with the given group and kind name.
func (m *Manifest) FindKind(group, kind string) (*Kind, bool) {
	for i := range m.Kinds {
		if m.Kinds[i].Group == group && m.Kinds[i].Kind == kind {
			return &m.Kinds[i], true
		}
	}

	return nil, false
}

// TypedKinds returns the kinds that have a typed Terraform resource.
func (m *Manifest) TypedKinds() []Kind {
	var out []Kind

	for _, k := range m.Kinds {
		if k.Typed {
			out = append(out, k)
		}
	}

	return out
}
