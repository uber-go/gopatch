package pgo

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/text"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		desc string
		give []byte
		want *File
	}{
		{
			desc: "top level expression",
			give: text.Unlines("foo()"),
			want: &File{
				Node: &Expr{
					Expr: &ast.CallExpr{
						Lparen: 3,
						Fun:    &ast.Ident{Name: "foo"},
						Rparen: 4,
					},
				},
			},
		},
		{
			desc: "statements",
			give: text.Unlines(
				"foo := x()",
				"if foo {",
				"  bar()",
				"}",
			),
			want: &File{
				Node: &StmtList{
					List: []ast.Stmt{
						// foo := x()
						&ast.AssignStmt{
							Lhs:    []ast.Expr{&ast.Ident{Name: "foo"}},
							Tok:    token.DEFINE,
							TokPos: 4,
							Rhs: []ast.Expr{
								&ast.CallExpr{
									Fun:    &ast.Ident{Name: "x", NamePos: 7},
									Lparen: 8,
									Rparen: 9,
								},
							},
						},
						&ast.IfStmt{
							// if foo
							If:   11,
							Cond: &ast.Ident{Name: "foo", NamePos: 14},
							Body: &ast.BlockStmt{
								Lbrace: 18,
								List: []ast.Stmt{
									// bar()
									&ast.ExprStmt{
										X: &ast.CallExpr{
											Fun:    &ast.Ident{Name: "bar", NamePos: 22},
											Lparen: 25,
											Rparen: 26,
										},
									},
								},
								Rbrace: 28,
							},
						},
					},
				},
			},
		},
		{
			desc: "top level function",
			give: text.Unlines(
				"func foo() {",
				"	bar()",
				"}",
			),
			want: &File{
				Node: &FuncDecl{
					FuncDecl: &ast.FuncDecl{
						Name: &ast.Ident{Name: "foo", NamePos: 5},
						Type: &ast.FuncType{
							Params: &ast.FieldList{
								Opening: 8,
								Closing: 9,
							},
						},
						Body: &ast.BlockStmt{
							Lbrace: 11,
							List: []ast.Stmt{
								&ast.ExprStmt{
									X: &ast.CallExpr{
										Fun:    &ast.Ident{Name: "bar", NamePos: 14},
										Lparen: 17,
										Rparen: 18,
									},
								},
							},
							Rbrace: 20,
						},
					},
				},
			},
		},
		{
			desc: "type declaration",
			give: text.Unlines(
				"type foo string",
			),
			want: &File{
				Node: &GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.TYPE,
						Specs: []ast.Spec{
							&ast.TypeSpec{
								Name: &ast.Ident{Name: "foo", NamePos: 5},
								Type: &ast.Ident{Name: "string", NamePos: 9},
							},
						},
					},
				},
			},
		},
		{
			desc: "package and expression",
			give: text.Unlines(
				"package foo",
				"",
				"bar()",
			),
			want: &File{
				Package: "foo",
				Node: &Expr{
					Expr: &ast.CallExpr{
						Fun:    &ast.Ident{Name: "bar", NamePos: 13},
						Lparen: 16,
						Rparen: 17,
					},
				},
			},
		},
		{
			desc: "statement list with dots",
			give: text.Unlines(
				`x := "foo"`,
				"...",
				`x += "bar"`,
			),
			want: &File{
				Node: &StmtList{
					List: []ast.Stmt{
						// x := "foo"
						&ast.AssignStmt{
							Lhs:    []ast.Expr{&ast.Ident{Name: "x"}},
							Tok:    token.DEFINE,
							TokPos: 2,
							Rhs: []ast.Expr{
								&ast.BasicLit{
									ValuePos: 5,
									Kind:     token.STRING,
									Value:    `"foo"`,
								},
							},
						},
						&ast.ExprStmt{X: &Dots{Dots: 11}},
						&ast.AssignStmt{
							Lhs:    []ast.Expr{&ast.Ident{Name: "x", NamePos: 15}},
							Tok:    token.ADD_ASSIGN,
							TokPos: 17,
							Rhs: []ast.Expr{
								&ast.BasicLit{
									ValuePos: 20,
									Kind:     token.STRING,
									Value:    `"bar"`,
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "dots inside struct",
			give: text.Unlines(
				"type foo struct {",
				"  ...",
				"}",
			),
			want: &File{
				Node: &GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.TYPE,
						Specs: []ast.Spec{
							&ast.TypeSpec{
								Name: &ast.Ident{Name: "foo", NamePos: 5},
								Type: &ast.StructType{
									Struct: 9,
									Fields: &ast.FieldList{
										Opening: 16,
										List: []*ast.Field{
											{Type: &Dots{Dots: 20}},
										},
										Closing: 24,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "dots as expression",
			give: text.Unlines("foo(...)"),
			want: &File{
				Node: &Expr{
					Expr: &ast.CallExpr{
						Fun:    &ast.Ident{Name: "foo"},
						Lparen: 3,
						Args: []ast.Expr{
							&Dots{Dots: 4},
						},
						Rparen: 7,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()
			got, err := Parse(fset, "test.go", tt.give)
			require.NoError(t, err)

			// To keep test cases simple, offset the positions by file.Base().
			file := fset.File(got.Node.Pos())
			assert.Equal(t, "test.go", file.Name())
			goast.OffsetPos(got.Node, -file.Base())

			disconnectIdentObj(reflect.ValueOf(got.Node))
			if !assert.Equal(t, tt.want, got) {
				// testify's diff doesn't handle recursive structures like
				// ast.Ident.
				pretty.Ldiff(t, tt.want, got)
			}
		})
	}
}

// Looks for ast.Idents and disconnects them from their ast.Objects.
// Otherwise we end up with a cyclic reference.
func disconnectIdentObj(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	if v.Type() == goast.IdentPtrType {
		ident := v.Interface().(*ast.Ident)
		ident.Obj = nil
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			disconnectIdentObj(v.Index(i))
		}
	case reflect.Interface, reflect.Ptr:
		disconnectIdentObj(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			disconnectIdentObj(v.Field(i))
		}
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		desc    string
		give    []byte
		wantErr string
	}{
		{
			desc:    "empty",
			wantErr: "test.go: expected a declaration or an expression, found EOF",
		},
		{
			desc: "no decl",
			give: text.Unlines(
				"package foo",
			),
			wantErr: "test.go:1:11: expected a declaration or an expression, found EOF",
		},
		{
			desc: "too many declarations",
			give: text.Unlines(
				"type foo struct{}",
				"type bar struct{}",
			),
			wantErr: "test.go:2:1: unexpected declaration: patches can have exactly one declaration",
		},
		{
			desc: "unexpected dots",
			give: text.Unlines(
				"type ... struct {}",
			),
			wantErr: `test.go:1:6: found unexpected "..." inside *ast.Ident`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := Parse(fset, "test.go", tt.give)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				ast.Print(nil, f)
			}
		})
	}
}
