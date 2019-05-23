package engine

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	"github.com/uber-go/gopatch/internal/goast"
	"github.com/stretchr/testify/assert"
)

func refl(v interface{}) reflect.Value {
	return reflect.ValueOf(v)
}

func stringPtr(s string) *string { return &s }

func TestGeneric(t *testing.T) {
	type emptyStruct struct{}
	type someNode struct{ X string }

	type matchCase struct {
		desc string        // name of the test case
		give reflect.Value // value to match
		ok   bool          // expected result
	}

	tests := []struct {
		name  string        // name of the matcher
		give  reflect.Value // value to compile
		cases []matchCase   // cases for this matcher
	}{
		{
			name: "string pointer",
			give: refl(stringPtr("foo")),
			cases: []matchCase{
				{
					desc: "string match",
					give: refl(stringPtr("foo")),
					ok:   true,
				},
				{
					desc: "string mismatch",
					give: refl(stringPtr("bar")),
				},
				{
					desc: "string nil",
					give: refl((*string)(nil)),
				},
			},
		},
		{
			name: "nil pointer",
			give: refl((*string)(nil)),
			cases: []matchCase{
				{
					desc: "nil",
					give: refl((*string)(nil)),
					ok:   true,
				},
				{
					desc: "non-nil",
					give: refl(stringPtr("")),
				},
			},
		},
		{
			name: "slice",
			give: refl([]int{1, 2, 3}),
			cases: []matchCase{
				{
					desc: "slice match",
					give: refl([]int{1, 2, 3}),
					ok:   true,
				},
				{
					desc: "slice mismatch",
					give: refl([]int{1, 3, 2}),
				},
				{
					desc: "slice nil",
					give: refl([]int(nil)),
				},
			},
		},
		{
			name: "nil slice",
			give: refl([]int(nil)),
			cases: []matchCase{
				{
					desc: "nil",
					give: refl([]int(nil)),
					ok:   true,
				},
				{
					desc: "non-nil",
					give: refl([]int{}),
				},
			},
		},
		{
			name: "empty struct",
			give: refl(struct{}{}),
			cases: []matchCase{
				{
					desc: "match",
					give: refl(struct{}{}),
					ok:   true,
				},
				{
					desc: "different struct with the same shape",
					give: refl(emptyStruct{}),
				},
				{
					desc: "different type",
					give: refl(42),
				},
			},
		},
		{
			name: "someNode struct",
			give: refl(someNode{X: "foo"}),
			cases: []matchCase{
				{
					desc: "match",
					give: refl(someNode{X: "foo"}),
					ok:   true,
				},
				{
					desc: "different value",
					give: refl(someNode{X: "bar"}),
				},
				{
					desc: "different type",
					give: refl(struct{}{}),
				},
				{
					desc: "similar shape",
					give: refl(struct{ X string }{X: "foo"}),
				},
			},
		},
		{
			name: "ast.Node interface",
			give: refl(&ast.BasicLit{Kind: token.INT, Value: "42"}).Convert(goast.NodeType),
			cases: []matchCase{
				{
					desc: "match",
					give: refl(&ast.BasicLit{Kind: token.INT, Value: "42"}).Convert(goast.NodeType),
					ok:   true,
				},
				{
					desc: "different interface same value",
					give: refl(&ast.BasicLit{Kind: token.INT, Value: "42"}).Convert(goast.ExprType),
					ok:   true,
					// Different interface with the same value is acceptable
					// because we want to be more liberal about matching
					// complex ASTs.
				},
				{
					desc: "different value",
					give: refl(&ast.BasicLit{Kind: token.INT, Value: "4"}).Convert(goast.NodeType),
				},
				{
					desc: "nil",
					give: reflect.Zero(goast.NodeType),
				},
			},
		},
		{
			name: "nil interface",
			give: reflect.Zero(goast.NodeType),
			cases: []matchCase{
				{
					desc: "nil",
					give: reflect.Zero(goast.NodeType),
					ok:   true,
				},
				{
					desc: "non-nil",
					give: refl(&ast.BasicLit{Kind: token.INT, Value: "42"}).Convert(goast.NodeType),
				},
			},
		},
		{
			name: "value",
			give: refl(42),
			cases: []matchCase{
				{
					desc: "match",
					give: refl(42),
					ok:   true,
				},
				{
					desc: "type mismatch",
					give: refl(int16(42)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMatcherCompiler().compileGeneric(tt.give)
			for _, tc := range tt.cases {
				t.Run(tc.desc, func(t *testing.T) {
					assert.Equal(t, tc.ok, m.Match(tc.give))
				})
			}
		})
	}
}
