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

package pgo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"

	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo/augment"
)

// Parse parses a pgo file. AST nodes in the returned File reference a newly
// added token.File in the given FileSet.
func Parse(fset *token.FileSet, filename string, src []byte) (*File, error) {
	pgoFile := fset.AddFile(filename, -1, len(src))
	pgoFile.SetLinesForContent(src)

	// TODO: The current system doesn't work for just top-level types.
	// For example, you can't parse just "[]T" with this because the generated
	// "func _fake() { []T }" is not valid Go code.
	//
	// One option is to use parser.ParseExpr for top-level expressions, in
	// which case augmentations, imports, etc. have to be handled separately.
	// So augment should probably return a File object with package name
	// already parsed, text for the unparsed imports, and the source. We can
	// then try ParseExpr on the source.

	src, augs, adjs, err := augment.Augment(src)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(byOffset(adjs)))

	// We'll use adjuster to map positions from the augmented file back to the
	// user-provided file.
	adjuster := posAdjuster{
		Fset: fset,
		File: pgoFile,
		Adjs: adjs,
	}

	f, err := parser.ParseFile(fset, filename+".augmented", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// f.Decls includes import declarations.
	f.Decls = filterImports(f.Decls)

	file := File{
		Imports:  f.Imports,
		Package:  f.Name.Name,
		Comments: f.Comments,
	}

	// FakePackage will always be the first augmentation if it's there. If we
	// have a FakePackage, we should ignore the parsed package name.
	if len(augs) > 0 {
		if _, isFakePackage := augs[0].(*augment.FakePackage); isFakePackage {
			file.Package = ""
			augs = augs[1:]
		}
	}

	// We allow only one declaration in the patch.
	switch len(f.Decls) {
	case 0:
		// This is impossible. Even for an empty string, we'll generate an
		// empty function which will translate to a declaration.
		panic("impossible: could not find a declaration")

	case 1:
		// This is what we expect.

	default:
		decl := f.Decls[1]
		return nil, fmt.Errorf("%v: unexpected declaration: "+
			"patches can have exactly one declaration", adjuster.Position(decl.Pos()))
	}

	var n ast.Node = f.Decls[0]
	// FakeFunc will always be the next augmentation if it's there. If we're
	// using a FakeFunc, the part of the AST under consideration in ast.Source
	// is the body of the function.
	if len(augs) > 0 {
		if _, isFakeFunc := augs[0].(*augment.FakeFunc); isFakeFunc {
			augs = augs[1:]
			body := n.(*ast.FuncDecl).Body
			n = body

			// If the body contains a single expression, it was from a
			// top-level expression.
			switch len(body.List) {
			case 0:
				// If the body is empty, the user didn't provide anything
				// after the package/imports.

				// TODO(abg): This should be the position after the package
				// and imports.
				return nil, fmt.Errorf("%v: expected a declaration or an expression, found EOF", adjuster.Position(body.Pos()))

				// TODO(abg): We should support zero declarations for
				// import/package-only transforms.
			case 1:
				// If the body contains a single expression, we want to do an
				// expression transformation.
				if es, ok := body.List[0].(*ast.ExprStmt); ok {
					n = es.X
				}
			}
		}
	}

	n, err = augmentAST(fset.File(n.Pos()), n, augs, &adjuster)
	if err != nil {
		return nil, err
	}

	var node Node
	switch n := n.(type) {
	case ast.Expr:
		node = &Expr{Expr: n}
	case *ast.BlockStmt:
		// TODO: check for empty BlockStmt
		node = &StmtList{List: n.List}
	case *ast.FuncDecl:
		node = &FuncDecl{FuncDecl: n}
	case *ast.GenDecl:
		node = &GenDecl{GenDecl: n}
	default:
		panic(fmt.Sprintf("impossible: unknown top-level type %T", n))
	}

	// We get back an AST with positions inside the .augmented file. Map them
	// back to the pgoFile we created above.
	goast.TransformPos(node, adjuster.Pos)

	file.Node = node
	return &file, nil
}

func filterImports(decls []ast.Decl) []ast.Decl {
	newDecls := decls[:0]
	for _, d := range decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			continue
		}
		newDecls = append(newDecls, d)
	}
	return newDecls
}

// posAdjuster maps positions in the augmented file back to the unaugmented
// patch file.
type posAdjuster struct {
	Fset *token.FileSet
	File *token.File
	Adjs []augment.PosAdjustment // sorted by offset
}

// Pos maps the provided position to the File associated with the Adjuster,
// using the provided PosAdjustments to offset it appropriately.
func (a *posAdjuster) Pos(pos token.Pos) token.Pos {
	off := a.Fset.File(pos).Offset(pos)

	i := sort.Search(len(a.Adjs), func(i int) bool {
		return a.Adjs[i].Offset <= off
	})
	if i < len(a.Adjs) {
		off -= a.Adjs[i].ReduceBy
	}
	if off < 0 {
		off = 0
	}

	return a.File.Pos(off)
}

// Position returns the full positional information for the given token.Pos
// after adjustment.
func (a *posAdjuster) Position(pos token.Pos) token.Position {
	return a.Fset.Position(a.Pos(pos))
}

type byOffset []augment.PosAdjustment

func (a byOffset) Len() int {
	return len(a)
}

func (a byOffset) Less(i int, j int) bool {
	return a[i].Offset < a[j].Offset
}

func (a byOffset) Swap(i int, j int) {
	a[i], a[j] = a[j], a[i]
}
