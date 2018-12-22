package goast

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOffsetPos(t *testing.T) {
	tests := []struct {
		desc   string
		give   interface{}
		offset int
		want   ast.Node
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			OffsetPos(tt.give, tt.offset)
			assert.Equal(t, tt.want, tt.give)
		})
	}
}

func TestOffsetPosPanicsOnMap(t *testing.T) {
	assert.Panics(t, func() {
		OffsetPos(struct{ M map[string]token.Pos }{}, 42)
	})
}
