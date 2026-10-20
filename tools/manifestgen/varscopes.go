//go:build wfmanifestgen

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// varScopeInventory discovers varsHandler constructors registered from init.
// Blind spot: scopes served by a different handler type are not detected.
func varScopeInventory(root string) ([]string, error) {
	dir := filepath.Join(root, "pkg/apiserver/handlers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("scan vars handlers: %w", err)
	}
	constructors := make(map[string][]*ast.CompositeLit)
	registered := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse vars handler %s: %w", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if _, ok := node.(*ast.FuncLit); ok {
					return false
				}
				if fn.Name.Name == "init" {
					call, ok := node.(*ast.CallExpr)
					if !ok || len(call.Args) != 1 {
						return true
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || selector.Sel.Name != "RegisterHandler" {
						return true
					}
					pkg, ok := selector.X.(*ast.Ident)
					if !ok || pkg.Name != "apiregister" {
						return true
					}
					constructor, ok := call.Args[0].(*ast.CallExpr)
					if !ok || len(constructor.Args) != 0 {
						return true
					}
					if name, ok := constructor.Fun.(*ast.Ident); ok {
						registered[name.Name] = true
					}
					return true
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
					if typ, ok := literal.Type.(*ast.Ident); ok && typ.Name == "varsHandler" {
						constructors[fn.Name.Name] = append(constructors[fn.Name.Name], literal)
					}
				}
				return false
			})
		}
	}
	var scopes []string
	for _, name := range slices.Sorted(maps.Keys(registered)) {
		for _, literal := range constructors[name] {
			var value ast.Expr
			for _, element := range literal.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := field.Key.(*ast.Ident); ok && key.Name == "scopeName" {
					value = field.Value
					break
				}
			}
			scope, ok := value.(*ast.BasicLit)
			if !ok || scope.Kind != token.STRING {
				return nil, fmt.Errorf("registered varsHandler constructor %s: scopeName must be a string literal", name)
			}
			unquoted, err := strconv.Unquote(scope.Value)
			if err != nil {
				return nil, fmt.Errorf("registered varsHandler constructor %s: unquote scopeName: %w", name, err)
			}
			scopes = append(scopes, unquoted)
		}
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("no registered varsHandler scopes found in pkg/apiserver/handlers; the vars API changed shape - update tools/manifestgen/varscopes.go")
	}
	slices.Sort(scopes)
	return slices.Compact(scopes), nil
}

// VarScopeCoverageError lists differences between kinds.yaml and the vars API.
type VarScopeCoverageError struct {
	Tag     string
	Missing []string // served but absent from kinds.yaml varScopes
	Stale   []string // listed in kinds.yaml varScopes but not served
}

func (e *VarScopeCoverageError) Error() string {
	var b strings.Builder
	b.WriteString("varScopes in kinds.yaml do not match the Wayfinder vars API:")
	for _, scope := range e.Missing {
		fmt.Fprintf(&b, "\n  - scope %q is served at %s but missing from kinds.yaml varScopes", scope, e.Tag)
	}
	for _, scope := range e.Stale {
		fmt.Fprintf(&b, "\n  - scope %q is listed in kinds.yaml varScopes but no longer served at %s", scope, e.Tag)
	}
	return b.String()
}

// CheckVarScopeCoverage fails unless kinds.yaml lists exactly the served scopes.
func CheckVarScopeCoverage(kinds KindsFile, discovered []string, tag string) error {
	listed := make(map[string]bool, len(kinds.VarScopes))
	for _, entry := range kinds.VarScopes {
		listed[entry.Scope] = true
	}
	served := make(map[string]bool, len(discovered))
	cerr := &VarScopeCoverageError{Tag: tag}
	for _, scope := range discovered {
		served[scope] = true
		if !listed[scope] {
			cerr.Missing = append(cerr.Missing, scope)
		}
	}
	for _, entry := range kinds.VarScopes {
		if !served[entry.Scope] {
			cerr.Stale = append(cerr.Stale, entry.Scope)
		}
	}
	if len(cerr.Missing) == 0 && len(cerr.Stale) == 0 {
		return nil
	}
	slices.Sort(cerr.Missing)
	slices.Sort(cerr.Stale)
	return cerr
}
