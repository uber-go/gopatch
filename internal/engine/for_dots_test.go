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

// Generates a ForStmt with the provided body and Init, Cond, or Post
// statements based on the provided booleans. If everything is true,
// generates,
//
//  for i ;= 0; i < 10; i++ { $body }
func forStmtWith(init, cond, post bool, body *ast.BlockStmt) *ast.ForStmt {
	stmt := &ast.ForStmt{Body: body}

	if init {
		stmt.Init = &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("i")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "0"},
			},
		}
	}

	if cond {
		stmt.Cond = &ast.BinaryExpr{
			X:  ast.NewIdent("i"),
			Op: token.LEQ,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
		}
	}

	if post {
		stmt.Post = &ast.IncDecStmt{
			X:   ast.NewIdent("i"),
			Tok: token.INC,
		}
	}

	return stmt
}

// Generates a RangeStmt with the provided body and Key and Value set based on
// the provided booleans. If everything is true, generates,
//
//  for i, v := range x { $body }
func rangeStmtWith(key, value bool, body *ast.BlockStmt) *ast.RangeStmt {
	stmt := &ast.RangeStmt{
		X:    ast.NewIdent("x"),
		Body: body,
	}

	if key {
		stmt.Key = ast.NewIdent("i")
	}

	if value {
		stmt.Value = ast.NewIdent("v")
	}

	return stmt
}

// Generates a "{ $name(i) }" BlockStmt.
func callFuncBlock(name string) *ast.BlockStmt {
	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: ast.NewIdent(name),
					Args: []ast.Expr{
						ast.NewIdent("i"),
					},
				},
			},
		},
	}
}

func TestForNoDots(t *testing.T) {
	// This tests that we recognize all non-dots combinations of init,
	// cond, and post.

	tests := []struct {
		name string
		init bool // whether init is set
		cond bool // whether cond is set
		post bool // whether post is set
	}{
		{"empty", false, false, false},
		{"init", true, false, false},
		{"cond", false, true, false},
		{"post", false, false, true},
		{"init_cond", true, true, false},
		{"init_post", true, false, true},
		{"cond_post", false, true, true},
		{"init_cond_post", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minus := forStmtWith(tt.init, tt.cond, tt.post, callFuncBlock("i"))
			plus := &ast.ForStmt{} // for {}

			fset := token.NewFileSet()

			m := newMatcherCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(minus))
			r := newReplacerCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(plus))

			d, ok := m.Match(reflect.ValueOf(minus), data.New(), Region{})
			require.True(t, ok, "must match self")

			got, err := r.Replace(d, NewChangelog(), 0)
			require.NoError(t, err, "replace must succeed")

			assert.Equal(t, plus, got.Interface())
		})
	}
}

func TestForDots(t *testing.T) {
	// Equivalent to,
	//   for ... { f(i) }
	minus := &ast.ForStmt{Cond: &pgo.Dots{}, Body: callFuncBlock("f")}

	// Equivalent to,
	//  for ... { g(i) }
	plus := &ast.ForStmt{Cond: &pgo.Dots{}, Body: callFuncBlock("g")}

	tests := []struct {
		desc string

		// Go AST to match against.
		give reflect.Value

		// Expected output from Replace in case of a match, and an
		// invalid reflect.Value for match failure.
		want reflect.Value
	}{
		{
			desc: "mismatch",
			// if i <= 10 { f(i) }
			give: refl(&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  ast.NewIdent("i"),
					Op: token.LEQ,
					Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
				},
				Body: callFuncBlock("f"),
			}),
		},
		{
			desc: "match",
			give: refl(&ast.ForStmt{
				Init: &ast.AssignStmt{
					Lhs: []ast.Expr{ast.NewIdent("x")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.BasicLit{Kind: token.INT, Value: "0"},
					},
				},
				Cond: &ast.BinaryExpr{
					X:  ast.NewIdent("x"),
					Op: token.LEQ,
					Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
				},
				Body: &ast.BlockStmt{},
			}),
		},

		// Validate that "for ..." handles all combinations of init, cond
		// and post in a ForStmt.

		{
			desc: "for/empty",
			give: refl(forStmtWith(false, false, false, callFuncBlock("f"))),
			want: refl(forStmtWith(false, false, false, callFuncBlock("g"))),
		},
		{
			desc: "for/init",
			give: refl(forStmtWith(true, false, false, callFuncBlock("f"))),
			want: refl(forStmtWith(true, false, false, callFuncBlock("g"))),
		},
		{
			desc: "for/cond",
			give: refl(forStmtWith(false, true, false, callFuncBlock("f"))),
			want: refl(forStmtWith(false, true, false, callFuncBlock("g"))),
		},
		{
			desc: "for/post",
			give: refl(forStmtWith(false, false, true, callFuncBlock("f"))),
			want: refl(forStmtWith(false, false, true, callFuncBlock("g"))),
		},
		{
			desc: "for/init_cond",
			give: refl(forStmtWith(true, true, false, callFuncBlock("f"))),
			want: refl(forStmtWith(true, true, false, callFuncBlock("g"))),
		},
		{
			desc: "for/init_post",
			give: refl(forStmtWith(true, false, true, callFuncBlock("f"))),
			want: refl(forStmtWith(true, false, true, callFuncBlock("g"))),
		},
		{
			desc: "for/cond_post",
			give: refl(forStmtWith(false, true, true, callFuncBlock("f"))),
			want: refl(forStmtWith(false, true, true, callFuncBlock("g"))),
		},
		{
			desc: "for/init_cond_post",
			give: refl(forStmtWith(true, true, true, callFuncBlock("f"))),
			want: refl(forStmtWith(true, true, true, callFuncBlock("g"))),
		},

		// Validates that "for ..." handles all combinations of key
		// and value in a RangeStmt.

		{
			desc: "range/empty",
			give: refl(rangeStmtWith(false, false, callFuncBlock("f"))),
			want: refl(rangeStmtWith(false, false, callFuncBlock("g"))),
		},
		{
			desc: "range/key",
			give: refl(rangeStmtWith(true, false, callFuncBlock("f"))),
			want: refl(rangeStmtWith(true, false, callFuncBlock("g"))),
		},
		{
			desc: "range/value",
			give: refl(rangeStmtWith(false, true, callFuncBlock("f"))),
			want: refl(rangeStmtWith(false, true, callFuncBlock("g"))),
		},
		{
			desc: "range/key_value",
			give: refl(rangeStmtWith(true, true, callFuncBlock("f"))),
			want: refl(rangeStmtWith(true, true, callFuncBlock("g"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()

			m := newMatcherCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(minus))
			r := newReplacerCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(plus))

			d, ok := m.Match(tt.give, data.New(), Region{})
			if !tt.want.IsValid() {
				require.False(t, ok, "did not expect a match")
				return
			}

			require.True(t, ok, "expected match success")

			got, err := r.Replace(d, NewChangelog(), 0)
			require.NoError(t, err)

			assert.Equal(t, tt.want.Interface(), got.Interface())
		})
	}
}

func TestForDotsNoMatchData(t *testing.T) {
	fset := token.NewFileSet()
	file := fset.AddFile("patch", -1, 100)

	// Equivalent to,
	//   for ... { f(i) }
	minus := &ast.ForStmt{Cond: &pgo.Dots{Dots: file.Pos(10)}, Body: callFuncBlock("f")}
	file.AddLineColumnInfo(10, "patch", 1, 1)

	// Equivalent to,
	//  for ... { g(i) }
	plus := &ast.ForStmt{Cond: &pgo.Dots{Dots: file.Pos(12)}, Body: callFuncBlock("g")}
	file.AddLineColumnInfo(12, "patch", 2, 1)

	// We've told the FileSet that the two "..."s are on different lines
	// in the patch so they won't be matched.

	m := newMatcherCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(minus))
	r := newReplacerCompiler(fset, nil /* meta */, 0, 0).compileForStmt(reflect.ValueOf(plus))

	d, ok := m.Match(reflect.ValueOf(forStmtWith(true, true, true, callFuncBlock("f"))), data.New(), Region{})
	require.True(t, ok, "must match")

	_, err := r.Replace(d, NewChangelog(), 0)
	require.Error(t, err, "must fail")
	assert.Contains(t, err.Error(), "match data not found")
}
