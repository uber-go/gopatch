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
	"fmt"
	"go/ast"
	"go/token"
	"sort"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/parse"
)

// Change is a single Change in a program.
type Change struct {
	Name string
	Meta *Meta

	Comments []string
	fset     *token.FileSet
	matcher  FileMatcher
	replacer FileReplacer
}

func (c *compiler) compileChange(achange *parse.Change) *Change {
	meta := c.compileMeta(achange.Meta)

	mc := newMatcherCompiler(c.fset, meta, achange.Patch.Pos(), achange.Patch.End())
	rc := newReplacerCompiler(c.fset, meta, achange.Patch.Pos(), achange.Patch.End())

	matcher := mc.compileFile(achange.Patch.Minus)
	replacer := rc.compileFile(achange.Patch.Plus)

	ldots := mc.dots
	rdots := rc.dots
	connectDots(c.fset, ldots, rdots, rc.dotAssoc)

	return &Change{
		Name:     achange.Name, // TODO(abg): validate name
		Meta:     meta,
		fset:     c.fset,
		matcher:  matcher,
		replacer: replacer,
		Comments: achange.Comments,
	}
}

// Match matches this change in the given Go AST and returns captured match
// information it a data.Data object.
func (c *Change) Match(f *ast.File) (d data.Data, ok bool) {
	return c.matcher.Match(f, data.New())
}

// Replace generates a replacement File based on previously captured match
// data.
func (c *Change) Replace(d data.Data, cl Changelog) (*ast.File, error) {
	return c.replacer.Replace(d, cl)
}

func connectDots(fset *token.FileSet, lhs, rhs []token.Pos, conns map[token.Pos]token.Pos) error {
	cache := make(map[token.Pos]token.Position)
	getPosition := func(pos token.Pos) token.Position {
		p, ok := cache[pos]
		if !ok {
			p = fset.Position(pos)
			cache[pos] = p
		}
		return p
	}

	sort.Slice(lhs, func(i, j int) bool {
		pi := getPosition(lhs[i])
		pj := getPosition(lhs[j])

		// Descending order.
		return pi.Line > pj.Line || pi.Line == pj.Line && pi.Column > pj.Column
	})

	sort.Slice(rhs, func(i, j int) bool {
		pi := getPosition(rhs[i])
		pj := getPosition(rhs[j])

		// Ascending order.
		return pi.Line < pj.Line || pi.Line == pj.Line && pi.Column < pj.Column
	})

	for _, r := range rhs {
		rpos := getPosition(r)

		i := sort.Search(len(lhs), func(i int) bool {
			lpos := getPosition(lhs[i])
			return lpos.Line < rpos.Line || lpos.Line == rpos.Line && lpos.Column <= rpos.Column
		})

		if i == len(lhs) {
			return fmt.Errorf(`%v: "..." in "+" section does not have an associated "..." in "-" section`, rpos)
		}

		if other, conflict := conns[r]; conflict {
			lpos := getPosition(lhs[i])
			conflictPos := getPosition(other)
			return fmt.Errorf(
				`cannot associate "..." at %v with %v: already associated with %v`,
				rpos, lpos, conflictPos,
			)
		}

		conns[r] = lhs[i]
	}

	return nil
}
