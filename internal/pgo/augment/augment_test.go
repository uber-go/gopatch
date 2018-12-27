package augment

import (
	"testing"

	"github.com/uber-go/gopatch/internal/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAugment(t *testing.T) {
	tests := []struct {
		desc     string
		give     []byte
		wantSrc  []byte
		wantAugs []Augmentation
		wantAdjs []PosAdjustment
	}{
		{
			desc: "with package name",
			give: text.Unlines(
				"package foo",
				"",
				"foo()",
			),
			wantSrc: text.Unlines(
				"package foo",
				"",
				"func _() {",
				"foo()",
				"}",
			),
			wantAugs: []Augmentation{
				&FakeFunc{FuncStart: 13, Braces: true},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 13, ReduceBy: 11},
			},
		},
		{
			desc: "imports/group",
			give: text.Unlines(
				"import (",
				`  "fmt"`,
				`  _ "net/http/pprof"`,
				`  goast "go/ast"`,
				"",
				`  . "somepackage"`,
				")",
				"",
				"x := 42",
			),
			wantSrc: text.Unlines(
				"package _",
				"import (",
				`  "fmt"`,
				`  _ "net/http/pprof"`,
				`  goast "go/ast"`,
				"",
				`  . "somepackage"`,
				")",
				"",
				"func _() {",
				"x := 42",
				"}",
			),
			wantAugs: []Augmentation{
				&FakePackage{PackageStart: 0},
				&FakeFunc{FuncStart: 87, Braces: true},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 0, ReduceBy: 10},
				{Offset: 87, ReduceBy: 21},
			},
		},
		{
			desc: "imports/dot",
			give: text.Unlines(
				"package foo",
				"",
				`import . "bar"`,
				"",
				"{",
				"  x += 1",
				"}",
			),
			wantSrc: text.Unlines(
				"package foo",
				"",
				`import . "bar"`,
				"",
				"func _() {",
				"  x += 1",
				"}",
			),
			wantAugs: []Augmentation{
				&FakeFunc{FuncStart: 29},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 29, ReduceBy: 9},
			},
		},
		{
			desc: "imports/underscore",
			give: text.Unlines(
				"package foo",
				"",
				`import _ "net/http/pprof"`,
				"",
				"func foo() {",
				"  x += 1",
				"}",
			),
			wantSrc: text.Unlines(
				"package foo",
				"",
				`import _ "net/http/pprof"`,
				"",
				"func foo() {",
				"  x += 1",
				"}",
			),
		},
		{
			desc: "imports/named",
			give: text.Unlines(
				`import bar "baz"`,
				"",
				"type Foo struct{}",
			),
			wantSrc: text.Unlines(
				"package _",
				`import bar "baz"`,
				"",
				"type Foo struct{}",
			),
			wantAugs: []Augmentation{
				&FakePackage{PackageStart: 0},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 0, ReduceBy: 10},
			},
		},
		{
			desc: "dots in statements",
			give: text.Unlines(
				"foo()",
				"...",
				"bar()",
			),
			wantSrc: text.Unlines(
				"package _",
				"func _() {",
				"foo()",
				"dts",
				"bar()",
				"}",
			),
			wantAugs: []Augmentation{
				&FakePackage{PackageStart: 0},
				&FakeFunc{FuncStart: 10, Braces: true},
				&Dots{DotsStart: 27, DotsEnd: 30},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 0, ReduceBy: 10},
				{Offset: 10, ReduceBy: 21},
			},
		},
		{
			desc: "dots in parameters",
			give: text.Unlines(
				"foo(bar, ..., baz)",
			),
			wantSrc: text.Unlines(
				"package _",
				"func _() {",
				"foo(bar, dts, baz)",
				"}",
			),
			wantAugs: []Augmentation{
				&FakePackage{PackageStart: 0},
				&FakeFunc{FuncStart: 10, Braces: true},
				&Dots{DotsStart: 30, DotsEnd: 33},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 0, ReduceBy: 10},
				{Offset: 10, ReduceBy: 21},
			},
		},
		{
			desc: "func with splats",
			give: text.Unlines(
				"func foo(args ...string) {",
				"  foo(args...)",
				"}",
			),
			wantSrc: text.Unlines(
				"package _",
				"func foo(args ...string) {",
				"  foo(args...)",
				"}",
			),
			wantAugs: []Augmentation{
				&FakePackage{PackageStart: 0},
			},
			wantAdjs: []PosAdjustment{
				{Offset: 0, ReduceBy: 10},
			},
		},
		{
			desc: "function signature with splat",
			give: text.Unlines(
				"package foo",
				"",
				"type Foo func(...string)",
			),
			wantSrc: text.Unlines(
				"package foo",
				"",
				"type Foo func(...string)",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotSrc, gotAugs, gotAdjs, err := Augment(tt.give)
			require.NoError(t, err)

			// Comparing strings instead of []bytes gives better error
			// messages with testify.
			assert.Equal(t, string(tt.wantSrc), string(gotSrc))
			assert.Equal(t, tt.wantAugs, gotAugs)
			assert.Equal(t, tt.wantAdjs, gotAdjs)
		})
	}
}
