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
	"github.com/uber-go/gopatch/internal/text"
)

func TestParseMeta(t *testing.T) {
	ident := func(pos token.Pos, name string) *ast.Ident {
		return &ast.Ident{Name: name, NamePos: pos}
	}

	tests := []struct {
		desc string
		give []byte

		// Name and index of the change.
		changeIdx  int
		changeName string

		// Positions in want should be offsets in give STARTING AT 1. 0 is an
		// invalid value for token.Pos. These values will be adjusted relative
		// to the file's Base() later.
		want Meta

		// error messages should start with line 2 because we prepend a "@@\n"
		// to give.
		wantErrs []string
	}{
		{
			desc: "empty",
			give: text.Unlines(),
			want: Meta{},
		},
		{
			desc: "single var",
			give: text.Unlines("var foo identifier"),
			want: Meta{
				Vars: []*VarDecl{
					{
						VarPos: 1,
						Names: []*ast.Ident{
							ident(5, "foo"),
						},
						Type: ident(9, "identifier"),
					},
				},
			},
		},
		{
			desc:       "multiple vars",
			give:       text.Unlines("var foo, bar, baz identifier"),
			changeName: "foo",
			want: Meta{
				Vars: []*VarDecl{
					{
						VarPos: 1,
						Names: []*ast.Ident{
							ident(5, "foo"),
							ident(10, "bar"),
							ident(15, "baz"),
						},
						Type: ident(19, "identifier"),
					},
				},
			},
		},
		{
			desc: "multiple decls",
			give: text.Unlines(
				"var foo identifier",
				"var bar, baz identifier",
				"var qux, quux expression",
			),
			want: Meta{
				Vars: []*VarDecl{
					{
						VarPos: 1,
						Names: []*ast.Ident{
							ident(5, "foo"),
						},
						Type: ident(9, "identifier"),
					},
					{
						VarPos: 20,
						Names: []*ast.Ident{
							ident(24, "bar"),
							ident(29, "baz"),
						},
						Type: ident(33, "identifier"),
					},
					{
						VarPos: 44,
						Names: []*ast.Ident{
							ident(48, "qux"),
							ident(53, "quux"),
						},
						Type: ident(58, "expression"),
					},
				},
			},
		},
		{
			desc: "variable without var",
			give: text.Unlines("xar identifier"),
			wantErrs: []string{
				`test.patch:2:1: unexpected "IDENT", expected "var"`,
			},
		},
		{
			desc: "too many idents",
			give: text.Unlines(
				"var x identifier",
				"var x y z",
			),
			wantErrs: []string{
				`test.patch:3:9: unexpected "IDENT", expected ";" or a newline`,
			},
		},
		{
			desc: "too many vars",
			give: text.Unlines(
				"var x identifier",
				"var var var",
				"var y expression",
			),
			wantErrs: []string{
				`test.patch:3:5: unexpected "var", expected an identifier`,
			},
		},
		{
			desc: "missing type name",
			give: text.Unlines("var x"),
			wantErrs: []string{
				`test.patch:2:6: unexpected ";", expected an identifier`,
			},
		},
		{
			desc: "unrecognized token",
			give: text.Unlines("var # foo"),
			wantErrs: []string{
				// This messages comes from go/scanner.
				`test.patch:2:5: illegal character U+0023 '#'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()

			// Generate a fake .patch file with this metavariables section.
			var src bytes.Buffer
			fmt.Fprintf(&src, "@%v@\n", tt.changeName)
			src.Write(tt.give)
			fmt.Fprintln(&src, "@@")

			sections, err := section.Split(fset, "test.patch", src.Bytes())
			require.NoError(t, err)
			require.Len(t, sections, 1, "expected exactly one change")

			parser := newParser(fset)
			got, err := parser.parseMeta(tt.changeIdx, sections[0])

			if len(tt.wantErrs) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrs {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err)

			// Add the Base() for the parsed file to the positions in the test
			// cases so that we can express the test cases relative to 1.
			if len(got.Vars) > 0 {
				file := fset.File(got.Vars[0].Pos())
				goast.OffsetPos(tt.want, file.Base()-1)
			}

			if !assert.Equal(t, &tt.want, got) {
				// Testify's diff doesn't fully show ast.Ident because of the
				// possibility of a cyclic reference via ast.Ident.Obj so
				// we'll rely on kr/pretty for the diff in case of test
				// failure.
				pretty.Ldiff(t, &tt.want, got)
			}
		})
	}
}
