package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
	"github.com/uber-go/gopatch/internal/text"
	"github.com/kr/pretty"
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
				{
					desc: "type name left alone",
					giveSrc: text.Unlines(
						"package a",
						"func b() { var foo int }",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() { var foo int }",
					),
				},
			},
		},
		{
			desc: "func decl",
			// -foo() {}
			// +bar() {}
			minus: &pgo.File{
				Node: &pgo.FuncDecl{
					FuncDecl: &ast.FuncDecl{
						Name: ast.NewIdent("foo"),
						Type: &ast.FuncType{
							Params: &ast.FieldList{},
						},
						Body: &ast.BlockStmt{},
					},
				},
			},
			plus: &pgo.File{
				Node: &pgo.FuncDecl{
					FuncDecl: &ast.FuncDecl{
						Name: ast.NewIdent("bar"),
						Type: &ast.FuncType{
							Params: &ast.FieldList{},
						},
						Body: &ast.BlockStmt{},
					},
				},
			},
			cases: []testCase{
				{
					desc: "match",
					giveSrc: text.Unlines(
						"package a",
						"func foo() {}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func bar() {}",
					),
				},
			},
		},
		{
			desc: "var decl",
			// -var foo string
			// +var foo, bar UUID
			minus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("foo")},
								Type:  ast.NewIdent("string"),
							},
						},
					},
				},
			},
			plus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{
									ast.NewIdent("foo"),
									ast.NewIdent("bar"),
								},
								Type: ast.NewIdent("UUID"),
							},
						},
					},
				},
			},
			cases: []testCase{
				{
					desc: "match",
					giveSrc: text.Unlines(
						"package a",
						"func foo() {",
						"	var foo string",
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func foo() {",
						"	var foo, bar UUID",
						"}",
					),
				},
			},
		},
		{
			desc: "const decl",
			// -const x = "42"
			// +const x = 42
			minus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.CONST,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("x")},
								Values: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.STRING,
										Value: `"42"`,
									},
								},
							},
						},
					},
				},
			},
			plus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.CONST,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{ast.NewIdent("x")},
								Values: []ast.Expr{
									&ast.BasicLit{
										Kind:  token.INT,
										Value: `42`,
									},
								},
							},
						},
					},
				},
			},
			cases: []testCase{
				{
					desc: "top level",
					giveSrc: text.Unlines(
						"package a",
						`const x = "42"`,
					),
					wantSrc: text.Unlines(
						"package a",
						`const x = 42`,
					),
				},
				{
					desc: "nested",
					giveSrc: text.Unlines(
						"package a",
						"func b() {",
						`	const x = "42"`,
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() {",
						`	const x = 42`,
						"}",
					),
				},
			},
		},
		{
			desc: "type decl",
			// -type Foo struct{ Bar }
			// +type Foo struct{}
			minus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.TYPE,
						Specs: []ast.Spec{
							&ast.TypeSpec{
								Name: ast.NewIdent("Foo"),
								Type: &ast.StructType{
									Fields: &ast.FieldList{
										List: []*ast.Field{
											{Type: ast.NewIdent("Bar")},
										},
									},
								},
							},
						},
					},
				},
			},
			plus: &pgo.File{
				Node: &pgo.GenDecl{
					GenDecl: &ast.GenDecl{
						Tok: token.TYPE,
						Specs: []ast.Spec{
							&ast.TypeSpec{
								Name: ast.NewIdent("Foo"),
								Type: &ast.StructType{
									Fields: &ast.FieldList{},
								},
							},
						},
					},
				},
			},
			cases: []testCase{
				{
					desc: "top level",
					giveSrc: text.Unlines(
						"package a",
						`type Foo struct{ Bar }`,
					),
					wantSrc: text.Unlines(
						"package a",
						`type Foo struct{}`,
					),
				},
				{
					desc: "nested",
					giveSrc: text.Unlines(
						"package a",
						"func b() {",
						`	type Foo struct{ Bar }`,
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func b() {",
						`	type Foo struct{}`,
						"}",
					),
				},
			},
		},
		{
			desc: "stmt list",
			// -var foo string
			//  foo = bar
			// +foo := bar
			minus: &pgo.File{
				Node: &pgo.StmtList{
					List: []ast.Stmt{
						&ast.DeclStmt{
							Decl: &ast.GenDecl{
								Tok: token.VAR,
								Specs: []ast.Spec{
									&ast.ValueSpec{
										Names: []*ast.Ident{ast.NewIdent("foo")},
										Type:  ast.NewIdent("string"),
									},
								},
							},
						},
						&ast.AssignStmt{
							Lhs: []ast.Expr{ast.NewIdent("foo")},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{ast.NewIdent("bar")},
						},
					},
				},
			},
			plus: &pgo.File{
				Node: &pgo.StmtList{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{ast.NewIdent("foo")},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{ast.NewIdent("bar")},
						},
					},
				},
			},
			cases: []testCase{
				{
					desc: "block",
					giveSrc: text.Unlines(
						"package a",
						"func x() {",
						"	if y() {",
						"		var foo string",
						"		foo = bar",
						"	}",
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func x() {",
						"	if y() {",
						"		foo := bar",
						"	}",
						"}",
					),
				},
				{
					desc: "switch",
					giveSrc: text.Unlines(
						"package a",
						"func x() {",
						"	switch y() {",
						"	case z:",
						"		var foo string",
						"		foo = bar",
						"	}",
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func x() {",
						"	switch y() {",
						"	case z:",
						"		foo := bar",
						"	}",
						"}",
					),
				},
				{
					desc: "select",
					giveSrc: text.Unlines(
						"package a",
						"func x(ctx context.Context) {",
						"	select {",
						"	case <-ctx.Done():",
						"		var foo string",
						"		foo = bar",
						"	}",
						"}",
					),
					wantSrc: text.Unlines(
						"package a",
						"func x(ctx context.Context) {",
						"	select {",
						"	case <-ctx.Done():",
						"		foo := bar",
						"	}",
						"}",
					),
				},
			},
		},
		{
			desc: "match package name",
			//  package foo
			//
			// -FooClient
			// +Client
			minus: &pgo.File{
				Package: "foo",
				Node:    &pgo.Expr{Expr: ast.NewIdent("FooClient")},
			},
			plus: &pgo.File{
				Package: "foo",
				Node:    &pgo.Expr{Expr: ast.NewIdent("Client")},
			},
			cases: []testCase{
				{
					desc: "success",
					giveSrc: text.Unlines(
						"package foo",
						"type FooClient struct{}",
					),
					wantSrc: text.Unlines(
						"package foo",
						"type Client struct{}",
					),
				},
				{
					desc: "package name mismatch",
					giveSrc: text.Unlines(
						"package fooclient",
						"type FooClient struct{}",
					),
				},
			},
		},
		{
			desc: "change package name",
			// -package foo
			// -package bar
			//
			//  Client
			minus: &pgo.File{
				Package: "foo",
				Node:    &pgo.Expr{Expr: ast.NewIdent("Client")},
			},
			plus: &pgo.File{
				Package: "bar",
				Node:    &pgo.Expr{Expr: ast.NewIdent("Client")},
			},
			cases: []testCase{
				{
					desc: "success",
					giveSrc: text.Unlines(
						"package foo",
						"type Client struct{}",
					),
					wantSrc: text.Unlines(
						"package bar",
						"type Client struct{}",
					),
				},
				{
					desc: "body mismatch",
					giveSrc: text.Unlines(
						"package foo",
						"type FooClient struct{}",
					),
				},
			},
		},
		{
			desc: "remove import",
			// -import "foo"
			//  bar
			minus: &pgo.File{
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{Kind: token.STRING, Value: `"foo"`},
					},
				},
				Node: &pgo.Expr{Expr: ast.NewIdent("bar")},
			},
			plus: &pgo.File{
				Node: &pgo.Expr{Expr: ast.NewIdent("bar")},
			},
			cases: []testCase{
				{
					desc: "success",
					giveSrc: text.Unlines(
						"package x",
						`import "foo"`,
						"func bar() { bar() }",
					),
					wantSrc: text.Unlines(
						"package x",
						"func bar() { bar() }",
					),
				},
				{
					desc: "named import",
					giveSrc: text.Unlines(
						"package x",
						`import baz "foo"`,
						"func bar() { bar() }",
					),
					wantSrc: text.Unlines(
						"package x",
						"func bar() { bar() }",
					),
				},
			},
		},
		{
			desc: "replace import",
			// -import "foo"
			// -import "bar"
			// -foo.Bar
			// +bar.Bar
			minus: &pgo.File{
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{Kind: token.STRING, Value: `"foo"`},
					},
				},
				Node: &pgo.Expr{
					Expr: &ast.SelectorExpr{X: ast.NewIdent("foo"), Sel: ast.NewIdent("Bar")},
				},
			},
			plus: &pgo.File{
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{Kind: token.STRING, Value: `"bar"`},
					},
				},
				Node: &pgo.Expr{
					Expr: &ast.SelectorExpr{X: ast.NewIdent("bar"), Sel: ast.NewIdent("Bar")},
				},
			},
			cases: []testCase{
				{
					desc: "success",
					giveSrc: text.Unlines(
						"package x",
						`import "foo"`,
						"func x() { foo.Bar() }",
					),
					wantSrc: text.Unlines(
						"package x",
						`import "bar"`,
						"func x() { bar.Bar() }",
					),
				},
				// TODO: If the patch had foo.Bar, and the
				// user made a named import with baz, we need
				// to match baz.Bar instead of foo.Bar.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()

			matcher := newMatcherCompiler(fset, nil /* meta */, 0, 0).compileFile(tt.minus)
			replacer := newReplacerCompiler(fset, nil /* meta */, 0, 0).compileFile(tt.plus)

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
					gotFile, err := replacer.Replace(d, NewChangelog())
					require.NoError(t, err)
					if !assert.Equal(t, wantFile, gotFile, "files did not match") {
						// Files didn't match so print
						// a more readable diff.
						pretty.Ldiff(t, wantFile, gotFile)
					}
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

	disconnectIdentObj(reflect.ValueOf(file))
	return file
}

// Looks for ast.Idents and disconnects them from their ast.Objects.
// Otherwise we end up with a cyclic reference.
func disconnectIdentObj(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	if v.Type() == goast.IdentPtrType && !v.IsNil() {
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
