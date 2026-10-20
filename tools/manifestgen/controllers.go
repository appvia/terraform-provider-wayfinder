// Copyright 2026 Appvia Ltd.
// SPDX-License-Identifier: Apache-2.0

//go:build wfmanifestgen

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func checkoutRoot() (string, error) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("locate generator source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return "", fmt.Errorf("locate Wayfinder checkout at %s: %w", root, err)
	}
	return root, nil
}

func controllerInventory(root string) (map[string]bool, error) {
	inventory := make(map[string]bool)
	for _, subtree := range []string{"server", "pkg/controllers"} {
		err := filepath.WalkDir(filepath.Join(root, subtree), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if !strings.Contains(filepath.ToSlash(rel), "/controllers/") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return fmt.Errorf("parse controller %s: %w", rel, err)
			}
			for _, decl := range file.Decls {
				method, ok := decl.(*ast.FuncDecl)
				if !ok || method.Recv == nil || method.Name.Name != "Reconciles" || method.Body == nil {
					continue
				}
				ast.Inspect(method.Body, func(node ast.Node) bool {
					if _, ok := node.(*ast.FuncLit); ok {
						return false
					}
					ret, ok := node.(*ast.ReturnStmt)
					if !ok {
						return true
					}
					for _, result := range ret.Results {
						if unary, ok := result.(*ast.UnaryExpr); ok && unary.Op == token.AND {
							result = unary.X
						}
						literal, ok := result.(*ast.CompositeLit)
						if !ok {
							continue
						}
						switch typ := literal.Type.(type) {
						case *ast.SelectorExpr:
							inventory[typ.Sel.Name] = true
						case *ast.Ident:
							inventory[typ.Name] = true
						}
					}
					return false
				})
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan %s controllers: %w", subtree, err)
		}
	}
	return inventory, nil
}
