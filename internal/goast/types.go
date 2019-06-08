package goast

import (
	"go/ast"
	"reflect"
)

// Reflected types of various AST nodes.
var (
	// Struct Pointers
	IdentPtrType = reflect.TypeOf((*ast.Ident)(nil))

	// Interfaces
	ExprType = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	NodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()
)
