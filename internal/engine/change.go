package engine

import (
	"go/ast"
	"go/token"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/parse"
)

// Change is a single Change in a program.
type Change struct {
	Name string
	Meta *Meta

	fset     *token.FileSet
	matcher  FileMatcher
	replacer FileReplacer
}

func (c *compiler) compileChange(achange *parse.Change) *Change {
	meta := c.compileMeta(achange.Meta)

	mc := newMatcherCompiler(c.fset, meta, achange.Patch.Pos(), achange.Patch.End())
	rc := newReplacerCompiler(c.fset, meta)

	matcher := mc.compileFile(achange.Patch.Minus)
	replacer := rc.compileFile(achange.Patch.Plus)
	return &Change{
		Name:     achange.Name, // TODO(abg): validate name
		Meta:     meta,
		fset:     c.fset,
		matcher:  matcher,
		replacer: replacer,
	}
}

// Match matches this change in the given Go AST and returns captured match
// information it a data.Data object.
func (c *Change) Match(f *ast.File) (d data.Data, ok bool) {
	return c.matcher.Match(f, data.New())
}

// Replace generates a replacement File based on previously captured match
// data.
func (c *Change) Replace(d data.Data) (*ast.File, error) {
	return c.replacer.Replace(d)
}
