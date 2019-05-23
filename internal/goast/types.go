package goast

import (
	"go/ast"
	"reflect"
)

// Reflected types of various AST nodes.
var (
	// Interfaces
	ExprType = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	NodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()
)
