package engine

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericReplacer(t *testing.T) {
	tests := []struct {
		desc  string
		value reflect.Value
	}{
		{
			desc:  "string pointer",
			value: refl(stringPtr("foo")),
		},
		{
			desc:  "nil pointer",
			value: refl((*string)(nil)),
		},
		{
			desc:  "slice",
			value: refl([]int{1, 2, 3}),
		},
		{
			desc:  "nil slice",
			value: refl([]string(nil)),
		},
		{
			desc:  "empty struct",
			value: refl(struct{}{}),
		},
		{
			desc:  "non-empty struct",
			value: refl(struct{ Foo string }{Foo: "bar"}),
		},
		{
			desc:  "ast.Node interface",
			value: refl(&ast.BasicLit{Kind: token.INT, Value: "42"}).Convert(goast.NodeType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r := newReplacerCompiler(token.NewFileSet(), nil).compileGeneric(tt.value)

			t.Run("equality", func(t *testing.T) {
				got, err := r.Replace(data.New())
				require.NoError(t, err)
				assert.Equal(t, tt.value.Interface(), got.Interface())
			})

			// A Matcher constructed from this value matches the output of a
			// replacer built with it.
			t.Run("matches self", func(t *testing.T) {
				m := newMatcherCompiler(token.NewFileSet(), nil).compileGeneric(tt.value)

				got, err := r.Replace(data.New())
				require.NoError(t, err)

				_, ok := m.Match(got, data.New())
				assert.True(t, ok)
			})
		})
	}
}
