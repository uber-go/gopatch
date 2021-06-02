// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
