package goast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/gopatch/internal/text"
)

func TestOffsetPos(t *testing.T) {
	type testCase struct {
		desc   string
		give   interface{}
		offset int
		want   ast.Node
	}

	tests := []testCase{
		{
			desc:   "identifier",
			give:   &ast.Ident{NamePos: 42, Name: "Foo"},
			offset: 5,
			want:   &ast.Ident{NamePos: 47, Name: "Foo"},
		},
		{
			desc: "block stmt",
			give: &ast.BlockStmt{
				Lbrace: 12,
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun:    &ast.Ident{NamePos: 14, Name: "Bar"},
							Lparen: 18,
							Args: []ast.Expr{
								&ast.Ident{NamePos: 19, Name: "foo"},
							},
							Rparen: 34,
						},
					},
				},
				Rbrace: 42,
			},
			offset: -12,
			want: &ast.BlockStmt{
				Lbrace: 0,
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun:    &ast.Ident{NamePos: 2, Name: "Bar"},
							Lparen: 6,
							Args: []ast.Expr{
								&ast.Ident{NamePos: 7, Name: "foo"},
							},
							Rparen: 22,
						},
					},
				},
				Rbrace: 30,
			},
		},
		func() (tt testCase) {
			tt.desc = "file with func"
			tt.give = parseFile(t,
				"package foo",
				"func foo() {}",
			)
			tt.offset = 3

			fooFuncObj := &ast.Object{
				Kind: ast.Fun,
				Name: "foo",
			}

			file := &ast.File{
				Package: 4,
				Name: &ast.Ident{
					Name:    "foo",
					NamePos: 12,
				},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{
							Name:    "foo",
							NamePos: 21,
							Obj:     fooFuncObj,
						},
						Type: &ast.FuncType{
							Func: 16,
							Params: &ast.FieldList{
								Opening: 24,
								Closing: 25,
							},
						},
						Body: &ast.BlockStmt{
							Lbrace: 27,
							Rbrace: 28,
						},
					},
				},
				Scope: &ast.Scope{
					Objects: map[string]*ast.Object{
						"foo": fooFuncObj,
					},
				},
			}
			tt.want = file
			// Equivalent to,
			//   parseFile(t, "   package foo", ...)
			// (Ofsetting everything by 3.)

			// Connect the cyclic reference.
			fooFuncObj.Decl = file.Decls[0]

			return
		}(),
		func() (tt testCase) {
			tt.desc = "file with import"
			tt.give = parseFile(t,
				"package foo",
				`import bar "bar.git"`,
			)
			tt.offset = 10

			barImport := &ast.ImportSpec{
				Name: &ast.Ident{
					Name:    "bar",
					NamePos: 30,
				},
				Path: &ast.BasicLit{
					Value:    `"bar.git"`,
					ValuePos: 34,
					Kind:     token.STRING,
				},
			}

			file := &ast.File{
				Package: 11,
				Name: &ast.Ident{
					Name:    "foo",
					NamePos: 19,
				},
				Decls: []ast.Decl{
					&ast.GenDecl{
						TokPos: 23,
						Tok:    token.IMPORT,
						Specs:  []ast.Spec{barImport},
					},
				},
				// file.Imports contains references to all
				// imports found in the file but the Decl is
				// authoritative.
				Imports: []*ast.ImportSpec{barImport},
				Scope: &ast.Scope{
					Objects: map[string]*ast.Object{},
				},
			}
			tt.want = file
			// Equivalent to,
			//   parseFile(t, "          package foo", ...)
			// (Offsetting everything by 10.)

			return
		}(),
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			OffsetPos(tt.give, tt.offset)
			if !assert.Equal(t, tt.want, tt.give) {
				// Handles cyclic references better than
				// testify.
				pretty.Ldiff(t, tt.want, tt.give)
			}
		})
	}
}

func TestOffsetPosPanicsOnMap(t *testing.T) {
	assert.Panics(t, func() {
		OffsetPos(struct{ M map[string]token.Pos }{}, 42)
	})
}

func parseFile(t *testing.T, lines ...string) *ast.File {
	src := text.Unlines(lines...)
	file, err := parser.ParseFile(token.NewFileSet(), t.Name(), src, parser.ParseComments)
	require.NoError(t, err, "failed to parse source")
	return file
}
