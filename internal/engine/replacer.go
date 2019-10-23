package engine

import (
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
)

// Replacer generates portions of the Go AST meant to replace sections matched
// by a Matcher. A Replacer is built from the "+" portion of a patch.
type Replacer interface {
	// Replace generates values for a Go AST provided prior match data.
	Replace(d data.Data) (v reflect.Value, err error)
}

// replacerCompiler compiles the "+" portion of a patch into a Replacer which
// will generate the portions to fill in the original AST.
type replacerCompiler struct {
	fset *token.FileSet
	meta *Meta // declared metavariables, if any
}

func newReplacerCompiler(fset *token.FileSet, meta *Meta) *replacerCompiler {
	return &replacerCompiler{
		fset: fset,
		meta: meta,
	}
}

func (c *replacerCompiler) compile(v reflect.Value) Replacer {
	switch v.Type() {
	case goast.IdentPtrType:
		return c.compileIdent(v)
	case goast.ForStmtPtrType:
		return c.compileForStmt(v)

	case goast.ObjectPtrType:
		// Ident.Obj forms a cycle, and it doesn't affect the output
		// of the formatter. We'll replace it with a nil pointer.
		return ZeroReplacer{Type: goast.ObjectPtrType}
	}
	return c.compileGeneric(v)
}

// ZeroReplacer replaces with the zero value of a type.
type ZeroReplacer struct{ Type reflect.Type }

// Replace replaces with a zero value.
func (r ZeroReplacer) Replace(data.Data) (reflect.Value, error) {
	return reflect.Zero(r.Type), nil
}
