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
	"go/ast"
	"go/token"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/parse/section"
	"github.com/uber-go/gopatch/internal/pgo"
	"github.com/uber-go/gopatch/internal/text"
)

// sectionFromLines builds a section.Section with the given lines. The
// position of each line is its offset in the string, starting at 1.
func sectionFromLines(lines ...string) section.Section {
	var s section.Section
	pos := 1
	for _, l := range lines {
		line := &section.Line{
			StartPos: token.Pos(pos),
			Text:     []byte(l + "\n"),
		}
		s = append(s, line)
		pos += len(line.Text)
	}
	return s
}

func sectionSize(s section.Section) int {
	size := 0
	for _, l := range s {
		size += len(l.Text)
	}
	return size
}

func TestParsePatch(t *testing.T) {
	tests := []struct {
		desc       string
		give       section.Section
		changeName string

		want *Patch
	}{
		{
			desc: "change expression",
			give: sectionFromLines(
				"-foo",
				"+bar",
			),
			want: &Patch{
				Minus: &pgo.File{
					Node: &pgo.Expr{
						Expr: &ast.Ident{Name: "foo"},
					},
				},
				Plus: &pgo.File{
					Node: &pgo.Expr{
						Expr: &ast.Ident{Name: "bar"},
					},
				},
			},
		},
		{
			desc:       "change spread across lines",
			changeName: "hello",
			give: sectionFromLines(
				" foo(",
				"-  x,",
				"+  y,",
				" )",
			),
			want: &Patch{
				Minus: &pgo.File{
					Node: &pgo.Expr{
						Expr: &ast.CallExpr{
							Fun:  &ast.Ident{Name: "foo"},
							Args: []ast.Expr{&ast.Ident{Name: "x"}},
						},
					},
				},
				Plus: &pgo.File{
					Node: &pgo.Expr{
						Expr: &ast.CallExpr{
							Fun:  &ast.Ident{Name: "foo"},
							Args: []ast.Expr{&ast.Ident{Name: "y"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()
			parser := newParser(fset)
			file := fset.AddFile("test.patch", -1, sectionSize(tt.give))
			got, err := parser.parsePatch(0, &section.Change{
				Name:      tt.changeName,
				HeaderPos: file.Pos(0),
				Patch:     tt.give,
			})
			require.NoError(t, err)

			// Zero out all positions because we don't care.
			goast.TransformPos(got, func(token.Pos) token.Pos { return token.NoPos })

			if !assert.Equal(t, tt.want, got) {
				// Testify's diff doesn't fully show ast.Ident because of the
				// possibility of a cyclic reference via ast.Ident.Obj so
				// we'll rely on kr/pretty for the diff in case of test
				// failure.
				pretty.Ldiff(t, tt.want, got)
			}
		})
	}
}

func TestSeparatePatch(t *testing.T) {
	// Build up a fake change.
	var buff bytes.Buffer
	fmt.Fprintln(&buff, "@@")
	fmt.Fprintln(&buff, "@@")

	buff.Write(text.Unlines(
		" foo",
		"-bar",
		"+baz",
		"qux",
	))

	contents := buff.Bytes()

	fset := token.NewFileSet()
	prog, err := section.Split(fset, "test.patch", contents)
	require.NoError(t, err, "failed to split sections")
	require.Len(t, prog, 1, "expected exactly one change")

	before, after := splitPatch(prog[0].Patch)

	assert.Equal(t,
		text.Unlines(" foo", "bar", "qux"), before.Contents,
		"before contents do not match")
	assert.Equal(t,
		text.Unlines(" foo", "baz", "qux"), after.Contents,
		"after contents do not match")

	// Checks that the line positions specified in
	assertLinesMatch := func(t *testing.T, v patchVersion) {
		// Number of line entries must match number of newlines.
		numLines := bytes.Count(v.Contents, []byte("\n"))
		if !assert.Len(t, v.Lines, numLines, "number of lines does not match") {
			return
		}

		for i, l := range v.Lines {
			wantLine := tillEOL(contents, fset.File(l.Pos).Offset(l.Pos))
			gotLine := tillEOL(v.Contents, l.Offset)

			assert.Equalf(t, string(wantLine), string(gotLine),
				"contents of line %d don't match", i)
		}
	}

	t.Run("before", func(t *testing.T) {
		assertLinesMatch(t, before)
	})

	t.Run("after", func(t *testing.T) {
		assertLinesMatch(t, after)
	})
}

// Returns the line in b starting at offset off.
func tillEOL(b []byte, off int) []byte {
	line := b[off:]
	if idx := bytes.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return line
}
