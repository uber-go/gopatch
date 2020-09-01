package engine

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
	"golang.org/x/tools/go/ast/astutil"
)

// FileMatcher matches Go files.
type FileMatcher struct {
	// Matches the package name, if any.
	Package string

	// Imports in the file.
	Imports []*ast.ImportSpec

	// Matches nodes in the file.
	NodeMatcher Matcher
}

func (c *matcherCompiler) compileFile(file *pgo.File) FileMatcher {
	var m Matcher
	switch n := file.Node.(type) {
	case *pgo.Expr:
		m = c.compile(reflect.ValueOf(n.Expr))
	case *pgo.GenDecl:
		m = c.compile(reflect.ValueOf(n.GenDecl))
	case *pgo.FuncDecl:
		m = c.compile(reflect.ValueOf(n.FuncDecl))
	case *pgo.StmtList:
		m = c.compilePGoStmtList(n)
	default:
		panic(fmt.Sprintf("unknown pgo node %T", file.Node))
	}

	return FileMatcher{
		Package:     file.Package,
		Imports:     file.Imports,
		NodeMatcher: m,
	}
}

// Match matches against the file, recording information about all matches
// found in it.
func (m FileMatcher) Match(file *ast.File, d data.Data) (data.Data, bool) {
	// Match package name.
	if m.Package != "" && m.Package != file.Name.Name {
		// TODO(abg): Use an identMatcher with a constraint.
		return d, false
	}

	// Match imports.
	importPaths := make(map[string]string)
	for _, want := range m.Imports {
		wantPath := goast.ImportPath(want)

		spec := goast.FindImportSpec(file, wantPath)
		if spec == nil {
			// Fail the match because the package wasn't imported.
			return d, false
		}

		// TODO(abg): If the patch had named imports, automatically
		// map foo.Bar in the patch to whatever the actual import was.
		importPaths[wantPath] = goast.ImportName(spec)
	}

	// To match the body, we use astutil.Apply which traverses the AST and
	// provides a replaceable pointer to each node so that we can rewrite
	// the AST in-place.
	var matches []*searchResult
	// TODO(abg): Support nil NodeMatcher for when a patch is matching on
	// just the package name or import paths.
	astutil.Apply(file, func(cursor *astutil.Cursor) bool {
		d := d // don't change outer d

		n := cursor.Node()
		if n == nil {
			return false
		}

		d, ok := m.NodeMatcher.Match(reflect.ValueOf(n), d, nodeRegion(n))
		if !ok {
			return true
		}

		matches = append(matches, &searchResult{
			parent: cursor.Parent(),
			name:   cursor.Name(),
			index:  cursor.Index(),
			data:   data.Index(d),
			region: nodeRegion(cursor.Node()),
		})

		return true // keep looking
	}, nil /* post func */)

	if len(matches) == 0 {
		return d, false
	}

	return data.WithValue(d, fileMatchKey, fileMatchData{
		File:    file,
		Imports: importPaths,
		Matches: matches,
	}), true
}

// FileReplacer replaces an ast.File.
type FileReplacer struct {
	Fset *token.FileSet

	// Package name to change to, if any.
	Package string

	// Imports in the file.
	Imports []*ast.ImportSpec

	// Replaces matched nodes in the file.
	NodeReplacer Replacer
}

func (c *replacerCompiler) compileFile(file *pgo.File) FileReplacer {
	var r Replacer
	switch n := file.Node.(type) {
	case *pgo.Expr:
		r = c.compile(reflect.ValueOf(n.Expr))
	case *pgo.GenDecl:
		r = c.compile(reflect.ValueOf(n.GenDecl))
	case *pgo.FuncDecl:
		r = c.compile(reflect.ValueOf(n.FuncDecl))
	case *pgo.StmtList:
		r = c.compilePGoStmtList(n)
	default:
		panic(fmt.Sprintf("unknown pgo node %T", file.Node))
	}

	return FileReplacer{
		Fset:         c.fset,
		Package:      file.Package,
		Imports:      file.Imports,
		NodeReplacer: r,
	}
}

// Replace replaces a file using the provided Match data.
func (r FileReplacer) Replace(d data.Data, cl Changelog) (*ast.File, error) {
	var fd fileMatchData
	if !data.Lookup(d, fileMatchKey, &fd) {
		return nil, errors.New("no file match data found")
	}

	file := fd.File
	if r.Package != "" {
		file.Name.Name = r.Package
	}

	// Add replaced imports.
	for _, imp := range r.Imports {
		astutil.AddNamedImport(r.Fset, file, goast.ImportName(imp), goast.ImportPath(imp))
	}

	for _, m := range fd.Matches {
		v := reflect.Indirect(reflect.ValueOf(m.parent)).FieldByName(m.name)
		if !v.IsValid() {
			// This is a bug in our code.
			panic(fmt.Sprintf("%q is not a field of %T", m.name, m.parent))
		}

		if m.index >= 0 {
			v = v.Index(m.index)
		}

		give, err := r.NodeReplacer.Replace(m.data, cl, m.region.Pos)
		if err != nil {
			return nil, err
		}

		// If the generated value isn't assignable to the target, the match
		// was too eager. For example, trying to place "foo.Bar"
		// (SelectorExpr) where only an identifier is allowed (in a variable
		// declaration name, for example).
		if give.Type().AssignableTo(v.Type()) {
			v.Set(give)
		}
	}

	// Remove other imports if unused.
	for imp, impName := range fd.Imports {
		// TODO: UsesImport isn't great. We should use go/packages
		// with NeedName LoadMode to determine if an identifier was
		// used.
		if !astutil.UsesImport(file, imp) {
			astutil.DeleteNamedImport(r.Fset, file, impName, imp)
		}
	}

	if len(file.Imports) == 0 {
		// The AST prefers that the list be nil when empty.
		file.Imports = nil
	}

	return file, nil
}

type _fileMatchKey struct{}

var fileMatchKey _fileMatchKey

// searchResult is a pointer to a replaceable node in a Go AST. This is a copy
// of the information contained in an astutil.Cursor. We need this type
// because astutil.Cursor becomes invalida after the astutil.Apply call.
type searchResult struct {
	// Parent object containing the matched node.
	parent ast.Node

	// Name of the field of the parent node referring to the matched node.
	name string

	// If the field is a slice, this is a non-negative number specfiying
	// the index of that slice that refers to the matched node.
	index int

	// Captured match data. This is needed by Replacers.
	data data.Data

	region Region
}

type fileMatchData struct {
	File    *ast.File
	Imports map[string]string // import path -> name (if any, or "" if unnamed)
	Matches []*searchResult
}
