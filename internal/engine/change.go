package engine

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/parse"
)

// Change is a single Change in a program.
type Change struct {
	Name string
	Meta *Meta

	fset     *token.FileSet
	matcher  Matcher
	replacer Replacer
}

func (c *compiler) compileChange(achange *parse.Change) *Change {
	meta := c.compileMeta(achange.Meta)
	mc := newMatcherCompiler()
	rc := newReplacerCompiler()
	return &Change{
		Name:     achange.Name, // TODO(abg): validate name
		Meta:     meta,
		fset:     c.fset,
		matcher:  mc.compile(reflect.ValueOf(achange.Patch.Minus)),
		replacer: rc.compile(reflect.ValueOf(achange.Patch.Plus)),
	}
}

// Match searches for this change in the given AST Node and its descendants.
func (c *Change) Match(f *ast.File) (ok bool) {
	return c.matcher.Match(reflect.ValueOf(f))
}
