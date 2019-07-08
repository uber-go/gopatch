package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
	"github.com/uber-go/gopatch/internal/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	type testCase struct {
		desc string

		// Input and output Go source. The source is not matched
		// exactly; only the ASTs are matched.
		//
		// If wantSrc is empty, we don't expect a match.
		giveSrc, wantSrc []byte
	}

	tests := []struct {
		desc string

		// Minus and plus sections of the patch.
		minus, plus *pgo.File

		cases []testCase
	}{
		{
			desc: "expression",
			// -foo
			// +bar
			minus: &pgo.File{
				Node: &pgo.Expr{Expr: ast.NewIdent("foo")},
			},
			plus: &pgo.File{
				Node: &pgo.Expr{Expr: ast.NewIdent("bar")},
			},
			cases: []testCase{
				{
					desc: "success",
					giveSrc: text.Unlines(
						"package a",
						"func b() { foo() }",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() { bar() }",
					),
				},
				{
					desc: "no matches",
					giveSrc: text.Unlines(
						"package a",
						"func b() { bar() }",
					),
				},
				{
					desc: "return value",
					giveSrc: text.Unlines(
						"package a",
						"func b() int { return foo }",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() int { return bar }",
					),
				},
			},
		},
		{
			desc: "add selector",
			// -foo
			// +foo.Bar
			minus: &pgo.File{
				Node: &pgo.Expr{Expr: ast.NewIdent("foo")},
			},
			plus: &pgo.File{
				Node: &pgo.Expr{
					Expr: &ast.SelectorExpr{
						X:   ast.NewIdent("foo"),
						Sel: ast.NewIdent("Bar"),
					},
				},
			},
			cases: []testCase{
				{
					desc: "succesS",
					giveSrc: text.Unlines(
						"package a",
						"func b() { foo() }",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() { foo.Bar() }",
					),
				},
				{
					desc: "named import left untouched",
					giveSrc: text.Unlines(
						"package a",
						`import foo "foo.git"`,
						"func b() { foo() }",
					),
					wantSrc: text.Unlines(
						"package a",
						`import foo "foo.git"`,
						"func b() { foo.Bar() }",
					),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()

			matcher := newMatcherCompiler(fset, nil /* meta */).compileFile(tt.minus)
			replacer := newReplacerCompiler(fset, nil /* meta */).compileFile(tt.plus)

			for _, tc := range tt.cases {
				t.Run(tc.desc, func(t *testing.T) {
					giveFile := parseFile(t, fset, tc.giveSrc)
					d, ok := matcher.Match(giveFile, data.New())

					if len(tc.wantSrc) == 0 {
						require.False(t, ok, "did not expect a match")
						return
					}
					require.True(t, ok, "expected a match")

					wantFile := parseFile(t, fset, tc.wantSrc)
					gotFile, err := replacer.Replace(d)
					require.NoError(t, err)
					assert.Equal(t, wantFile, gotFile, "files did not match")
				})
			}
		})
	}
}

func parseFile(t *testing.T, fset *token.FileSet, src []byte) *ast.File {
	file, err := parser.ParseFile(fset, t.Name(), src, 0)
	require.NoError(t, err, "failed to parse source")

	// Disconnect Scope objects and Unresolved lists because they have no
	// bearing on the AST.
	file.Scope = nil
	file.Unresolved = nil

	// Invalidate token.Pos values in the ASTs because we don't care about
	// them right now.
	goast.TransformPos(file, func(token.Pos) token.Pos {
		return token.NoPos
	})

	return file
}
