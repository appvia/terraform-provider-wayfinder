// tfgen emits literal Terraform schemas and examples from the pinned manifest.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appvia/terraform-provider-wayfinder/internal/manifest"
)

type options struct{ out, examples string }

func main() {
	input := flag.String("manifest", "schema/wayfinder-manifest.json", "manifest path relative to repository root")
	out := flag.String("out", "internal/generated", "generated Go output directory")
	examples := flag.String("examples", "examples", "example output directory")
	flag.Parse()
	root, err := repositoryRoot()
	if err == nil {
		var m *manifest.Manifest
		m, err = manifest.Load(resolve(root, *input))
		if err == nil {
			err = generate(m, options{resolve(root, *out), resolve(root, *examples)})
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tfgen:", err)
		os.Exit(1)
	}
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository go.mod not found")
		}
		dir = parent
	}
}

func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
