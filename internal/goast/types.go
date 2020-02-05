package goast

import (
	"go/ast"
	"go/token"
	"reflect"
)

// Reflected types of various AST nodes.
var (
	// Primitives
	PosType    = reflect.TypeOf(token.Pos(0))
	StringType = reflect.TypeOf("")

	// Structs
	BlockStmtType    = reflect.TypeOf(ast.BlockStmt{})
	CaseClauseType   = reflect.TypeOf(ast.CaseClause{})
	CommClauseType   = reflect.TypeOf(ast.CommClause{})
	CommentGroupType = reflect.TypeOf(ast.CommentGroup{})
	FieldListType    = reflect.TypeOf(ast.FieldList{})
	FieldType        = reflect.TypeOf(ast.Field{})
	FileType         = reflect.TypeOf(ast.File{})
	ForStmtType      = reflect.TypeOf(ast.ForStmt{})
	FuncDeclType     = reflect.TypeOf(ast.FuncDecl{})
	GenDeclType      = reflect.TypeOf(ast.GenDecl{})
	IdentType        = reflect.TypeOf(ast.Ident{})
	ObjectType       = reflect.TypeOf(ast.Object{})
	RangeStmtType    = reflect.TypeOf(ast.RangeStmt{})
	ScopeType        = reflect.TypeOf(ast.Scope{})

	// Struct Pointers
	CommentGroupPtrType = reflect.PtrTo(CommentGroupType)
	FieldListPtrType    = reflect.PtrTo(FieldListType)
	FieldPtrType        = reflect.PtrTo(FieldType)
	FilePtrType         = reflect.PtrTo(FileType)
	ForStmtPtrType      = reflect.PtrTo(ForStmtType)
	FuncDeclPtrType     = reflect.PtrTo(FuncDeclType)
	GenDeclPtrType      = reflect.PtrTo(GenDeclType)
	IdentPtrType        = reflect.PtrTo(IdentType)
	ObjectPtrType       = reflect.PtrTo(ObjectType)
	RangeStmtPtrType    = reflect.PtrTo(RangeStmtType)
	ScopePtrType        = reflect.PtrTo(ScopeType)

	// Interfaces
	ExprType = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	NodeType = reflect.TypeOf((*ast.Node)(nil)).Elem()
	StmtType = reflect.TypeOf((*ast.Stmt)(nil)).Elem()

	// Slices
	ExprSliceType     = reflect.SliceOf(ExprType)
	FieldPtrSliceType = reflect.SliceOf(FieldPtrType)
	StmtSliceType     = reflect.SliceOf(StmtType)
)
