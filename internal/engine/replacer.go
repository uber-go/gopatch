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
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
)

// Replacer generates portions of the Go AST meant to replace sections matched
// by a Matcher. A Replacer is built from the "+" portion of a patch.
type Replacer interface {
	// Replace generates values for a Go AST provided prior match data.
	Replace(d data.Data, cl Changelog, pos token.Pos) (v reflect.Value, err error)
}

// replacerCompiler compiles the "+" portion of a patch into a Replacer which
// will generate the portions to fill in the original AST.
type replacerCompiler struct {
	fset     *token.FileSet
	meta     *Meta
	dots     []token.Pos
	dotAssoc map[token.Pos]token.Pos

	patchStart, patchEnd token.Pos
}

func newReplacerCompiler(fset *token.FileSet, meta *Meta, patchStart, patchEnd token.Pos) *replacerCompiler {
	return &replacerCompiler{
		fset:       fset,
		meta:       meta,
		dotAssoc:   make(map[token.Pos]token.Pos),
		patchStart: patchStart,
		patchEnd:   patchEnd,
	}
}

func (c *replacerCompiler) compile(v reflect.Value) Replacer {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return ZeroReplacer{Type: v.Type()}
	}

	switch v.Type() {
	case goast.IdentPtrType:
		return c.compileIdent(v)
	case goast.StmtSliceType:
		return c.compileSliceDots(v, func(n ast.Node) bool {
			es, ok := n.(*ast.ExprStmt)
			if ok {
				_, ok = es.X.(*pgo.Dots)
			}
			return ok
		})
	case goast.ExprSliceType:
		return c.compileSliceDots(v, func(n ast.Node) bool {
			_, ok := n.(*pgo.Dots)
			return ok
		})
	case goast.FieldPtrSliceType:
		// TODO(abg): pgo.Parse should probably replace this with a DotsField.
		return c.compileSliceDots(v, func(n ast.Node) bool {
			f, ok := n.(*ast.Field)
			if ok {
				_, ok = f.Type.(*pgo.Dots)
			}
			return ok
		})
	case goast.FieldListPtrType:
		return c.compileFieldList(v)
	case goast.ForStmtPtrType:
		return c.compileForStmt(v)
	case goast.CommentGroupPtrType:
		// TODO: We're currently ignoring comments in the replacement patch.
		// We should probably record them and report them in the top-level
		// file.
		return ValueReplacer{
			Value: reflect.ValueOf((*ast.CommentGroup)(nil)),
		}

	case goast.ObjectPtrType:
		// Ident.Obj forms a cycle so we'll replace it with a nil pointer.
		return ValueReplacer{
			Value: reflect.ValueOf((*ast.Object)(nil)),
		}

	case goast.PosType:
		return c.compilePosReplacer(v)
	}

	return c.compileGeneric(v)
}

// ZeroReplacer replaces with a zero value.
type ZeroReplacer struct{ Type reflect.Type }

// Replace replaces with a zero value.
func (r ZeroReplacer) Replace(data.Data, Changelog, token.Pos) (reflect.Value, error) {
	return reflect.Zero(r.Type), nil
}
