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
func Sprint(x interface{}) string {
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
