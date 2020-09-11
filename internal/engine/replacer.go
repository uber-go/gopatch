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
	case goast.ForStmtPtrType:
		return c.compileForStmt(v)

	case goast.ObjectPtrType:
		// Ident.Obj forms a cycle, and it doesn't affect the output
		// of the formatter. We'll replace it with a nil pointer.
		return ZeroReplacer{Type: goast.ObjectPtrType}
	case goast.PosType:
		return c.compilePosReplacer(v)
	}

	return c.compileGeneric(v)
}

// ZeroReplacer replaces with the zero value of a type.
type ZeroReplacer struct{ Type reflect.Type }

// Replace replaces with a zero value.
func (r ZeroReplacer) Replace(data.Data, Changelog, token.Pos) (reflect.Value, error) {
	return reflect.Zero(r.Type), nil
}
