package pgo

import "reflect"

// Reflected types of various PGo AST nodes.
var (
	FilePtrType     = reflect.TypeOf((*File)(nil))
	ExprPtrType     = reflect.TypeOf((*Expr)(nil))
	GenDeclPtrType  = reflect.TypeOf((*GenDecl)(nil))
	FuncDeclPtrType = reflect.TypeOf((*FuncDecl)(nil))
	StmtListPtrType = reflect.TypeOf((*StmtList)(nil))
)
