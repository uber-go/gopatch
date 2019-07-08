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
	BlockStmtType    = reflect.TypeOf(ast.BlockStmt{})
	CaseClauseType   = reflect.TypeOf(ast.CaseClause{})
	CommClauseType   = reflect.TypeOf(ast.CommClause{})
	CommentGroupType = reflect.TypeOf(ast.CommentGroup{})
	IdentType        = reflect.TypeOf(ast.Ident{})
	ObjectType       = reflect.TypeOf(ast.Object{})

	// Struct Pointers
	CommentGroupPtrType = reflect.PtrTo(CommentGroupType)
	IdentPtrType        = reflect.PtrTo(IdentType)
	ObjectPtrType       = reflect.PtrTo(ObjectType)

	// Interfaces
	ExprType = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	NodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()
)
