package engine

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
)

func sprint(x interface{}) string {
	switch v := x.(type) {
	case Matcher:
		return fmt.Sprintf("%T", v)
	case reflect.Value:
		return sprint(v.Interface())
	case []reflect.Value:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = sprint(item)
		}
		return fmt.Sprint(items)
	case *ast.Field:
		return sprint(&ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{v}}})
	case ast.Node:
		var out bytes.Buffer
		printer.Fprint(&out, token.NewFileSet(), v)
		return out.String()
	case []ast.Stmt:
		return sprint(&ast.BlockStmt{List: v})
	case []ast.Expr:
		return sprint(&ast.CompositeLit{Elts: v})
	default:
		return fmt.Sprint(x)
	}
}
