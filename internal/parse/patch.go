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

package parse

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"io"

	"github.com/uber-go/gopatch/internal/parse/section"
	"github.com/uber-go/gopatch/internal/pgo"
)

// parsePatch parses a Patch from the given source.
func (p *parser) parsePatch(i int, c *section.Change) (*Patch, error) {
	if len(c.Patch) == 0 {
		return nil, p.errf(c.AtPos, "invalid change: patch cannot be empty")
	}

	patch := Patch{
		StartPos: c.Patch[0].Pos(),
		EndPos:   c.End(),
	}

	filename := p.fset.File(c.Pos()).Name()
	if c.Name != "" {
		filename += "/" + c.Name
	} else {
		filename += fmt.Sprintf("/%d", i)
	}

	minus, plus := splitPatch(c.Patch)

	var err error
	patch.Minus, err = p.parsePatchVersion(filename+".minus", minus)
	if err != nil {
		return nil, err
	}

	patch.Plus, err = p.parsePatchVersion(filename+".plus", plus)
	if err != nil {
		return nil, err
	}

	// FIXME: Hack: If one of minus and plus believes their side is an
	// statement and the other believes it's an expression, make them both
	// expresisons.
	switch m := patch.Minus.Node.(type) {
	case *pgo.Expr:
		if _, ok := patch.Plus.Node.(*pgo.StmtList); ok {
			patch.Minus.Node = &pgo.StmtList{
				List: []goast.Stmt{
					&goast.ExprStmt{X: m.Expr},
				},
			}
		}
	case *pgo.StmtList:
		if p, ok := patch.Plus.Node.(*pgo.Expr); ok {
			patch.Plus.Node = &pgo.StmtList{
				List: []goast.Stmt{
					&goast.ExprStmt{X: p.Expr},
				},
			}
		}
	}

	return &patch, nil
}

// parses one version of the unified diff of a file.
func (p *parser) parsePatchVersion(name string, f patchVersion) (*pgo.File, error) {
	pfile, err := pgo.Parse(p.fset, name, f.Contents)
	if err != nil {
		return nil, err
	}

	// TODO: fill position data for lines before msrc.Node.
	file := p.fset.File(pfile.Node.Pos())
	for _, i := range f.Lines {
		position := p.fset.Position(i.Pos)
		file.AddLineColumnInfo(i.Offset, position.Filename, position.Line, position.Column)
	}

	return pfile, nil
}

// patchVersion is one of the versions of a patch specified in a unified diff.
type patchVersion struct {
	// Contents of the file.
	Contents []byte

	// Positional information for each line in Contents.
	//
	// Each LinePos contains matches an offset in Contents to a token.Pos in
	// the original patch file.
	Lines []section.LinePos
}

// splitPatch splits a patch into the before and after versions of the
// file.
//
// Given the unified diff,
//
//   foo
//  -bar
//  +baz
//   qux
//
// This functions splits it into,
//
//  Before  After
//  ------  -----
//  foo     foo
//  bar     baz
//  qux     qux
func splitPatch(patch section.Section) (before, after patchVersion) {
	var (
		minus, plus bytes.Buffer
		both        = io.MultiWriter(&minus, &plus)

		minusLines, plusLines []section.LinePos

		newline = []byte("\n")
	)

	for _, line := range patch {
		// If true, the corresponding item won't get a LinePos entry because
		// it wasn't written to this time.
		var skipMinus, skipPlus bool

		w := both
		if len(line.Text) > 0 {
			switch line.Text[0] {
			case '-':
				skipPlus = true
				w = &minus
				line.Text = line.Text[1:]
				line.StartPos++ // '-'

			case '+':
				skipMinus = true
				w = &plus
				line.Text = line.Text[1:]
				line.StartPos++ // '+'
			}
		}

		if !skipMinus {
			minusLines = append(minusLines, section.LinePos{
				Offset: minus.Len(),
				Pos:    line.StartPos,
			})
		}

		if !skipPlus {
			plusLines = append(plusLines, section.LinePos{
				Offset: plus.Len(),
				Pos:    line.StartPos,
			})
		}

		w.Write(line.Text)
		w.Write(newline)
	}

	return patchVersion{
			Contents: minus.Bytes(),
			Lines:    minusLines,
		}, patchVersion{
			Contents: plus.Bytes(),
			Lines:    plusLines,
		}
}
