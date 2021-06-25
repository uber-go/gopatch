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
	NameS         string  // TODO: better naming
	Name          Matcher // named import, if any
	NameIsMetavar bool    // records wehther the named import was a metavariable
	// TODO: This is janky.

	Path string // import path as a string
}

func (c *matcherCompiler) compileImport(imp *ast.ImportSpec) ImportMatcher {
	var (
		name          Matcher
		nameS         string
		nameIsMetavar bool
	)
	if n := imp.Name; n != nil {
		nameS = n.Name
		name = c.compileIdent(reflect.ValueOf(n))
		nameIsMetavar = c.meta.LookupVar(nameS) == IdentMetavarType
	}

	// TODO: In the future, we should try to determine the package name
	// automatically if the named import is not provided. This is
	// expensive because we'd have to resolve the package name from the
	// import path, so the code would have to actually exist on-disk.

	return ImportMatcher{
		NameS:         nameS,
		Name:          name,
		Path:          goast.ImportPath(imp),
		NameIsMetavar: nameIsMetavar,
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

	// We need to account for four cases here:
	//
	// +--------------+-------------+-----------------------+
	// | Patch import | File import | Behavior              |
	// +--------------+-------------+-----------------------+
	// | unnamed      | unnamed     | match                 |
	// | unnamed      | named       | no match              |
	// | named        | unnamed     | match if metavariable |
	// | named        | named       | match name            |
	// +--------------+-------------+-----------------------+

	// Patch import is unnamed. Match only if the file import is also
	// unnamed.
	if m.Name == nil {
		return d, spec.Name == nil
	}

	if spec.Name == nil {
		if !m.NameIsMetavar {
			// If the patch import is not a metavar, then we meant
			// to match it verbatim, so this is not a match.
			return d, false
		}

		// If the patch import is a metavar, then record a fake name
		// for the metavar so that there's a value associated with
		// "foo" in "foo.X", but also record that the real name for
		// this import metavar is empty.
		d = data.WithValue(d, importMetavarKey(m.NameS), importMetavarData{
			Unnamed: true,
		})

		d = data.WithValue(d, importKey(m.Path), importData{
			Name:       m.NameS,
			MetavarKey: importMetavarKey(m.NameS),
		})

		d, ok = m.Name.Match(reflect.ValueOf(&ast.Ident{
			NamePos: spec.Pos(),
			Name:    m.NameS,
		}), d, nodeRegion(spec))
	} else {
		d = data.WithValue(d, importKey(m.Path), importData{Name: spec.Name.Name})

		// Both are named. Match as-is and also associate the package
		// name with the import path so that we can delete it later.
		d, ok = m.Name.Match(reflect.ValueOf(spec.Name), d, nodeRegion(spec))
	}

	// Both imports are named. Match the name and move on.
	return d, ok
}

type importMetavarKey string

type importMetavarData struct{ Unnamed bool }

type importKey string // import path

type importData struct {
	Name string // package name of the import

	// Name of the import metavar we used to match this value, if any.
	MetavarKey importMetavarKey
}

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
	Name          Replacer // named import, if any
	NameS         string
	NameIsMetavar bool

	Path string // import path as a string
	Fset *token.FileSet
}

func (c *replacerCompiler) compileImport(imp *ast.ImportSpec) ImportReplacer {
	var (
		name          Replacer
		nameS         string
		nameIsMetavar bool
	)
	if n := imp.Name; n != nil {
		nameS = n.Name
		name = c.compileIdent(reflect.ValueOf(n))
		nameIsMetavar = c.meta.LookupVar(nameS) == IdentMetavarType
	}

	// TODO: Same as matcher, maybe we should attempt to determine the
	// package name.

	return ImportReplacer{
		Name:          name,
		NameS:         nameS,
		Path:          goast.ImportPath(imp),
		Fset:          c.fset,
		NameIsMetavar: nameIsMetavar,
	}
}

// Replace adds a single import. Returns the name of the import that was
// added.
func (r ImportReplacer) Replace(d data.Data, cl Changelog, f *ast.File) (string, error) {
	// name is the name we want to use for the named import, and pkgName is
	// how the rest of the file references this import.
	var name, pkgName string
	if r.Name != nil {
		// The name replacer will produce the value for the named
		// import specified in the patch as-is, or if it was a
		// metavariable, using the matched value.
		//
		// This is undesirable for the case where the named import
		// matched an unnamed import. For that case, we want to ignore
		// the recorded name. So, if the named import is a
		// metavariable that matched an unnamed import, don't look up
		// its recorded value.

		var mdata importMetavarData
		if r.NameIsMetavar {
			data.Lookup(d, importMetavarKey(r.NameS), &mdata)
			pkgName = r.NameS // default to metavar name
		}

		if !mdata.Unnamed {
			namev, err := r.Name.Replace(d, cl, f.Pos()) // pos is irrelevant
			if err != nil {
				return "", err
			}
			name = namev.Interface().(*ast.Ident).Name
			pkgName = name
		}

	} else {
		// TODO: more sophisticated package name guessing logic here
		// and below.
		pkgName = filepath.Base(r.Path)
	}

	if !astutil.AddNamedImport(r.Fset, f, name, r.Path) {
		return "", nil
	}
	return pkgName, nil
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
		var importName, pkgName string

		if idata := new(importData); data.Lookup(d, importKey(imp), idata) {
			pkgName = idata.Name
			importName = idata.Name

			// If we used a metavariable to match this import,
			// record if its name was actually empty.
			if mdata := new(importMetavarData); data.Lookup(d, idata.MetavarKey, mdata) {
				if mdata.Unnamed {
					importName = ""
				}
			}
		}

		if len(pkgName) == 0 {
			pkgName = filepath.Base(imp)
		}

		// If this import was replaced by an added import, kill it.
		_, replaced := taken[pkgName]
		if replaced || !usesNameAsTopLevel(f, pkgName) {
			astutil.DeleteNamedImport(r.Fset, f, importName, imp)
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
