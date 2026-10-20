// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

func generate(m *manifest.Manifest, o options) error {
	files, err := render(m)
	if err != nil {
		return err
	}

	if err := removeStale(files, o); err != nil {
		return err
	}

	for _, name := range slices.Sorted(maps.Keys(files)) {
		path := filepath.Join(o.out, name)
		if !strings.HasPrefix(name, "zz_") {
			path = filepath.Join(o.examples, name)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		mode := fs.FileMode(0644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0755
		}

		if err := os.WriteFile(path, files[name], mode); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}

func removeStale(files map[string][]byte, o options) error {
	for _, root := range []string{o.out, o.examples} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}

			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			name, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			name = filepath.ToSlash(name)
			if _, keep := files[name]; keep {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if isGenerated(data) {
				return os.Remove(path)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// isGenerated reports whether the file's leading comment block carries the tfgen marker, in Go/HCL
// (//) or shell (#) form.
func isGenerated(data []byte) bool {
	for line := range bytes.Lines(data[:min(len(data), 512)]) {
		line = bytes.TrimRight(line, "\n")
		if string(line) == "// "+generatedMarker || string(line) == "# "+generatedMarker {
			return true
		}
	}

	return false
}
