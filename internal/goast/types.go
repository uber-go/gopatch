package goast

import (
	"go/ast"
	"go/token"
	"reflect"
)

// Reflected types of various AST nodes.
var (
	// Primitives
	PosType = reflect.TypeOf(token.Pos(0))

	// Structs
	IdentType  = reflect.TypeOf(ast.Ident{})
	ObjectType = reflect.TypeOf(ast.Object{})

	// Struct Pointers
	IdentPtrType  = reflect.PtrTo(IdentType)
	ObjectPtrType = reflect.PtrTo(ObjectType)

	// Interfaces
	ExprType = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	NodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()
)
