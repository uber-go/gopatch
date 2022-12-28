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

package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
)

// Sprint is used primarily for debugging and prints a readable representation
// of the provided value.
func Sprint(x any) string {
	switch v := x.(type) {
	case Matcher:
		return fmt.Sprintf("%T", v)
	case reflect.Value:
		return Sprint(v.Interface())
	case []reflect.Value:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = Sprint(item)
		}
		return fmt.Sprint(items)
	case *ast.Field:
		return Sprint(&ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{v}}})
	case ast.Node:
		var out bytes.Buffer
		printer.Fprint(&out, token.NewFileSet(), v)
		return out.String()
	case []ast.Stmt:
		return Sprint(&ast.BlockStmt{List: v})
	case []ast.Expr:
		return Sprint(&ast.CompositeLit{Elts: v})
	default:
		return fmt.Sprint(x)
	}
}
