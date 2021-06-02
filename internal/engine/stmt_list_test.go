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
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/pgo"
)

func TestStmtSliceContainer(t *testing.T) {
	type testCase struct {
		desc string
		give reflect.Value

		want reflect.Value // zero value if no match is expected
	}

	tests := []struct {
		desc        string
		minus, plus *pgo.StmtList
		cases       []testCase
	}{
		{
			desc: "simple",
			// -foo
			// -bar
			// +baz
			minus: &pgo.StmtList{
				List: []ast.Stmt{
					&ast.ExprStmt{X: ast.NewIdent("foo")},
					&ast.ExprStmt{X: ast.NewIdent("bar")},
				},
			},
			plus: &pgo.StmtList{
				List: []ast.Stmt{
					&ast.ExprStmt{X: ast.NewIdent("baz")},
				},
			},
			cases: []testCase{
				{
					desc: "block match",
					// equivalent to,
					//   {
					//     foo
					//     bar
					//   }
					give: refl(&ast.BlockStmt{
						Lbrace: 1,
						List: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("foo")},
							&ast.ExprStmt{X: ast.NewIdent("bar")},
						},
						Rbrace: 10,
					}),
					want: refl(&ast.BlockStmt{
						Lbrace: 1,
						List: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("baz")},
						},
						Rbrace: 10,
					}),
				},
				{
					desc: "case clause",
					// equivalent to,
					//   case x:
					//     foo
					//     bar
					give: refl(&ast.CaseClause{
						Case:  1,
						List:  []ast.Expr{ast.NewIdent("x")},
						Colon: 7,
						Body: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("foo")},
							&ast.ExprStmt{X: ast.NewIdent("bar")},
						},
					}),
					want: refl(&ast.CaseClause{
						Case:  1,
						List:  []ast.Expr{ast.NewIdent("x")},
						Colon: 7,
						Body: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("baz")},
						},
					}),
				},
				{
					desc: "comm clause",
					// equivalent to,
					//   case x <- y:
					//     foo
					//     bar
					give: refl(&ast.CommClause{
						Case: 1,
						Comm: &ast.SendStmt{
							Chan:  ast.NewIdent("x"),
							Arrow: 7,
							Value: ast.NewIdent("y"),
						},
						Colon: 10,
						Body: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("foo")},
							&ast.ExprStmt{X: ast.NewIdent("bar")},
						},
					}),
					want: refl(&ast.CommClause{
						Case: 1,
						Comm: &ast.SendStmt{
							Chan:  ast.NewIdent("x"),
							Arrow: 7,
							Value: ast.NewIdent("y"),
						},
						Colon: 10,
						Body: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("baz")},
						},
					}),
				},
				{
					desc: "block mismatch",
					give: refl(&ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("bar")},
						},
					}),
				},
				{
					desc: "not a pointer",
					give: refl(ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ExprStmt{X: ast.NewIdent("foo")},
							&ast.ExprStmt{X: ast.NewIdent("bar")},
						},
					}),
				},
				{
					desc: "other type",
					// equivalent to,
					//   var foo = bar
					give: refl(&ast.DeclStmt{
						Decl: &ast.GenDecl{
							Tok: token.VAR,
							Specs: []ast.Spec{
								&ast.ValueSpec{
									Names: []*ast.Ident{
										ast.NewIdent("foo"),
									},
									Values: []ast.Expr{
										ast.NewIdent("bar"),
									},
								},
							},
						},
					}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()
			matcher := newMatcherCompiler(fset, nil, 0, 0).compilePGoStmtList(tt.minus)
			replacer := newReplacerCompiler(fset, nil, 0, 0).compilePGoStmtList(tt.plus)

			for _, tc := range tt.cases {
				t.Run(tc.desc, func(t *testing.T) {
					d := data.New()

					wantMatch := tc.want.IsValid()

					var ok bool
					d, ok = matcher.Match(tc.give, d, Region{})
					assert.Equal(t, wantMatch, ok, "unexpected match status")

					if !wantMatch {
						return
					}

					got, err := replacer.Replace(d, NewChangelog(), 0)
					require.NoError(t, err)
					assert.Equal(t, tc.want.Interface(), got.Interface(),
						"replaced value did not match")
				})
			}
		})
	}
}

func TestStmtSliceContainerReplacerNoData(t *testing.T) {
	repl := newReplacerCompiler(token.NewFileSet(), nil, 0, 0).compilePGoStmtList(&pgo.StmtList{
		List: []ast.Stmt{
			&ast.ExprStmt{X: ast.NewIdent("baz")},
		},
	})
	_, err := repl.Replace(data.New(), NewChangelog(), 0)
	require.Error(t, err)
}
