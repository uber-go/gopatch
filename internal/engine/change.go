package engine

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
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

// Match matches this change in the given Go AST and returns captured match
// information it a Data object.
func (c *Change) Match(f *ast.File) (d data.Data, ok bool) {
	return c.matcher.Match(reflect.ValueOf(f), data.New())
}

// Replace generates a replacement File based on previously captured match
// data.
func (c *Change) Replace(d data.Data) (*ast.File, error) {
	v, err := c.replacer.Replace(d)
	if err != nil {
		return nil, err
	}
	return v.Interface().(*ast.File), nil
}
