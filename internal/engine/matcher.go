package engine

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
)

// Region denotes the portion of the code being matched, i.e. the start and end
// position of the given Node.
type Region struct{ Pos, End token.Pos }

// nodeRegion returns the Region occupied by a given node.
func nodeRegion(n ast.Node) Region {
	return Region{Pos: n.Pos(), End: n.End()}
}

// Matcher matches values in a Go AST. It is built from the "-" portion of a
// patch.
type Matcher interface {
	// Match is called with the a value from the AST being compared with the
	// match data captured so far.
	//
	// Match reports whether the match succeeded and if so, returns the
	// original or a different Data object containing additional match data.
	Match(reflect.Value, data.Data, Region) (_ data.Data, ok bool)
}

// matcherCompiler compiles the "-" portion of a patch into a Matcher which
// will report whether another Go AST matches it.
type matcherCompiler struct {
	fset *token.FileSet
	meta *Meta

	// All dots found during match compilation.
	dots []token.Pos

	patchStart, patchEnd token.Pos
}

func newMatcherCompiler(fset *token.FileSet, meta *Meta, patchStart, patchEnd token.Pos) *matcherCompiler {
	return &matcherCompiler{
		fset:       fset,
		meta:       meta,
		patchStart: patchStart,
		patchEnd:   patchEnd,
	}
}

func (c *matcherCompiler) compile(v reflect.Value) Matcher {
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

		// TODO: Dedupe
	case goast.CommentGroupPtrType:
		// Comments shouldn't affect match.
		return successMatcher
	case goast.ObjectPtrType:
		// Ident.Obj forms a cycle. We'll consider Object pointers to always
		// match because the entites they point to will be matched separately
		// anyway.
		return successMatcher
	case goast.PosType:
		return c.compilePosMatcher(v)
	}

	return c.compileGeneric(v)
}

type matcherFunc func(reflect.Value) bool

func (f matcherFunc) Match(v reflect.Value, d data.Data, _ Region) (data.Data, bool) {
	return d, f(v)
}

var (
	// nilMatcher is a Matcher that only matches nil values.
	nilMatcher Matcher = matcherFunc(func(got reflect.Value) bool { return got.IsNil() })

	// successMatcher always return true.
	successMatcher Matcher = matcherFunc(func(reflect.Value) bool { return true })
)
