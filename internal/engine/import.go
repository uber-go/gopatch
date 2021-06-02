// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package engine

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"golang.org/x/tools/go/ast/astutil"
)

// ImportMatcher matches a single import.
type ImportMatcher struct {
	NameS string  // TODO: better naming
	Name  Matcher // named import, if any
	Path  string  // import path as a string
}

func (c *matcherCompiler) compileImport(imp *ast.ImportSpec) ImportMatcher {
	var (
		name  Matcher
		nameS string
	)
	if n := imp.Name; n != nil {
		nameS = n.Name
		name = c.compileIdent(reflect.ValueOf(n))
	}

	// TODO: In the future, we should try to determine the package name
	// automatically if the named import is not provided. This is
	// expensive because we'd have to resolve the package name from the
	// import path, so the code would have to actually exist on-disk.

	return ImportMatcher{
		NameS: nameS,
		Name:  name,
		Path:  goast.ImportPath(imp),
	}
}

// Match matches an import in a file. If the matcher was built with a named
// import where the name is a metavariable, then this will match both, named
// and unnamed imports and record that information in the patch data.
func (m ImportMatcher) Match(file *ast.File, d data.Data) (_ data.Data, ok bool) {
	spec := goast.FindImportSpec(file, m.Path)
	if spec == nil {
		return d, false
	}

	name := spec.Name
	if name != nil {
		// If we matched a named import, record the name.
		d = data.WithValue(d, importKey(m.Path), importData{Name: name.Name})
	}

	// Match was a success if this wasn't a named import.
	if m.Name == nil {
		return d, true
	}

	// if we didn't match a named import, assume that the name specified
	// in the patch is the name.
	if name == nil {
		name = &ast.Ident{
			NamePos: spec.Pos(),
			Name:    m.NameS,
		}
	}

	return m.Name.Match(reflect.ValueOf(name), d, nodeRegion(spec))
}

type importKey string // import path

type importData struct{ Name string }

// ImportsMatcher matches multiple imports in a Go file.
type ImportsMatcher struct {
	Imports []ImportMatcher
}

func (c *matcherCompiler) compileImports(imps []*ast.ImportSpec) ImportsMatcher {
	var ms []ImportMatcher
	for _, imp := range imps {
		ms = append(ms, c.compileImport(imp))
	}
	return ImportsMatcher{Imports: ms}
}

// Match matches a block of imports in a file.
func (m ImportsMatcher) Match(file *ast.File, d data.Data) (_ data.Data, ok bool) {
	matchedImports := make([]string, 0, len(m.Imports))
	for _, m := range m.Imports {
		d, ok = m.Match(file, d)
		if !ok {
			return d, false
		}

		matchedImports = append(matchedImports, m.Path)
	}

	return data.WithValue(d, importsKey, importsData{
		MatchedImports: matchedImports,
	}), true
}

type _importsKey string

var importsKey _importsKey

type importsData struct {
	MatchedImports []string // import paths
}

// ImportReplacer replaces imports in a file.
type ImportReplacer struct {
	Name Replacer // named import, if any
	Path string   // import path as a string
	Fset *token.FileSet
}

func (c *replacerCompiler) compileImport(imp *ast.ImportSpec) ImportReplacer {
	var name Replacer
	if imp.Name != nil {
		name = c.compileIdent(reflect.ValueOf(imp.Name))
	}

	// TODO: Same as matcher, maybe we should attempt to determine the
	// package name.

	return ImportReplacer{
		Name: name,
		Path: goast.ImportPath(imp),
		Fset: c.fset,
	}
}

// Replace adds a single import. Returns the name of the import that was
// added.
func (r ImportReplacer) Replace(d data.Data, cl Changelog, f *ast.File) (string, error) {
	var name string
	if r.Name != nil {
		namev, err := r.Name.Replace(d, cl, f.Pos()) // pos is irrelevant
		if err != nil {
			return "", err
		}
		name = namev.Interface().(*ast.Ident).Name
	}

	if !astutil.AddNamedImport(r.Fset, f, name, r.Path) {
		return "", nil
	}
	return name, nil
}

// ImportsReplacer replaces a block of imports.
type ImportsReplacer struct {
	Imports []ImportReplacer
	Fset    *token.FileSet
}

func (c *replacerCompiler) compileImports(imps []*ast.ImportSpec) ImportsReplacer {
	var rs []ImportReplacer
	for _, imp := range imps {
		rs = append(rs, c.compileImport(imp))
	}
	return ImportsReplacer{Imports: rs, Fset: c.fset}
}

// Replace adds zero or more imports t a file.
//
// Returns a list of the names of the imports that were added, if known.
func (r ImportsReplacer) Replace(d data.Data, cl Changelog, f *ast.File) ([]string, error) {
	var names []string
	for _, imp := range r.Imports {
		name, err := imp.Replace(d, cl, f)
		if err != nil {
			return nil, err
		}

		if len(name) > 0 {
			names = append(names, name)
		}
	}

	return names, nil
}

// Cleanup cleans up unused imports. newNames is a list of names of imports
// that were added by the plus sections.
func (r ImportsReplacer) Cleanup(d data.Data, f *ast.File, newNames []string) error {
	var impData importsData
	if !data.Lookup(d, importsKey, &impData) {
		return nil
	}

	taken := make(map[string]struct{})
	for _, n := range newNames {
		taken[n] = struct{}{}
	}

	// Delete matched imports that are no longer used.
	for _, imp := range impData.MatchedImports {
		var idata importData
		data.Lookup(d, importKey(imp), &idata)
		impName := idata.Name
		if len(impName) == 0 {
			impName = filepath.Base(imp)
		}

		// If this import was replaced by an added import, kill it.
		_, replaced := taken[impName]
		if replaced || !usesNameAsTopLevel(f, impName) {
			astutil.DeleteNamedImport(r.Fset, f, idata.Name, imp)
		}
	}

	// For each import decl, if the import is the last in the group,
	// delete the parens around it.
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.IMPORT {
			break
		}

		if len(d.Specs) == 1 && d.Lparen.IsValid() {
			d.Lparen = token.NoPos
			d.Rparen = token.NoPos
		}
	}

	// To make the AST happy, if f.Imports is empty, explicitly nil it.
	if len(f.Imports) == 0 {
		f.Imports = nil
	}

	return nil
}

// TODO: This is probably not the best place or method to implement this.
func usesNameAsTopLevel(f *ast.File, name string) bool {
	var used bool
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true // keep looking
		}

		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true // keep looking
		}

		if x.Name == name && x.Obj == nil {
			used = true
		}

		return false
	})
	return used
}
