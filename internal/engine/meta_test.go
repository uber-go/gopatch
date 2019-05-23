package engine

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/uber-go/gopatch/internal/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMeta(t *testing.T) {
	tests := []struct {
		desc string
		give *parse.Meta
		want map[string]MetavarType
	}{
		{
			desc: "empty",
			give: &parse.Meta{},
			want: map[string]MetavarType{
				"foo": 0, // unknown
			},
		},
		{
			desc: "identifier",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo identifier
						Names: []*ast.Ident{ast.NewIdent("foo")},
						Type:  ast.NewIdent("identifier"),
					},
					{
						// var bar, baz identifier
						Names: []*ast.Ident{
							ast.NewIdent("bar"),
							ast.NewIdent("baz"),
						},
						Type: ast.NewIdent("identifier"),
					},
				},
			},
			want: map[string]MetavarType{
				"foo": IdentMetavarType,
				"bar": IdentMetavarType,
				"baz": IdentMetavarType,
				"qux": 0, // unknown
			},
		},
		{
			desc: "expression",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo, bar expression
						Names: []*ast.Ident{
							ast.NewIdent("foo"),
							ast.NewIdent("bar"),
						},
						Type: ast.NewIdent("expression"),
					},
				},
			},
			want: map[string]MetavarType{
				"foo": ExprMetavarType,
				"bar": ExprMetavarType,
				"baz": 0, // unknown
			},
		},
		{
			desc: "mix",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo, bar identifier
						Names: []*ast.Ident{
							ast.NewIdent("foo"),
							ast.NewIdent("bar"),
						},
						Type: ast.NewIdent("identifier"),
					},
					{
						// var baz expression
						Names: []*ast.Ident{
							ast.NewIdent("baz"),
						},
						Type: ast.NewIdent("expression"),
					},
				},
			},
			want: map[string]MetavarType{
				"foo": IdentMetavarType,
				"bar": IdentMetavarType,
				"baz": ExprMetavarType,
				"qux": 0, // unknown
			},
		},
		{
			desc: "underscore",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo, _ identifier
						Names: []*ast.Ident{
							ast.NewIdent("foo"),
							ast.NewIdent("_"),
						},
						Type: ast.NewIdent("identifier"),
					},
					{
						// var _, bar expression
						Names: []*ast.Ident{
							ast.NewIdent("_"),
							ast.NewIdent("bar"),
						},
						Type: ast.NewIdent("expression"),
					},
				},
			},
			want: map[string]MetavarType{
				"foo": IdentMetavarType,
				"bar": ExprMetavarType,
				"qux": 0, // unknown
				"_":   0, // unknown
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c := newCompiler(token.NewFileSet())
			meta := c.compileMeta(tt.give)
			require.NoError(t, c.Err())

			for name, wantType := range tt.want {
				t.Run(name, func(t *testing.T) {
					gotType := meta.LookupVar(name)
					assert.Equal(t, wantType, gotType)
				})
			}
		})
	}
}

func TestCompileMetaErrors(t *testing.T) {
	tests := []struct {
		desc    string
		give    *parse.Meta
		wantErr string
	}{
		{
			desc: "unknown type",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo whateven
						Names: []*ast.Ident{ast.NewIdent("foo")},
						Type:  ast.NewIdent("whateven"),
					},
				},
			},
			wantErr: `unknown metavariable type "whateven"`,
		},
		{
			desc: "name conflict/same type",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo, bar, foo expression
						Names: []*ast.Ident{
							ast.NewIdent("foo"),
							ast.NewIdent("bar"),
							ast.NewIdent("foo"),
						},
						Type: ast.NewIdent("expression"),
					},
				},
			},
			wantErr: `cannot define metavariable "foo": name already taken by metavariable defined at`,
		},
		{
			desc: "name conflict/different type",
			give: &parse.Meta{
				Vars: []*parse.VarDecl{
					{
						// var foo expression
						Names: []*ast.Ident{ast.NewIdent("foo")},
						Type:  ast.NewIdent("expression"),
					},
					{
						// var foo identifier
						Names: []*ast.Ident{ast.NewIdent("foo")},
						Type:  ast.NewIdent("identifier"),
					},
				},
			},
			wantErr: `cannot define metavariable "foo": name already taken by metavariable defined at`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			c := newCompiler(token.NewFileSet())
			m := c.compileMeta(tt.give)
			err := c.Err()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)

			assert.NotPanics(t, func() { m.LookupVar("foo") })
		})
	}
}
