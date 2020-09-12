package pgo

import (
	"fmt"
	"go/ast"
)

// Visitor visits each Node in a pgo AST.
type Visitor = ast.Visitor

// Walk walks the given AST Node with the given visitor.
//
// This is a version of "go/ast".Walk that works with pgo nodes.
func Walk(v Visitor, n ast.Node) {
	ast.Walk(pgoVisitor{v}, n)
}

type pgoVisitor struct{ visitor Visitor }

func (v pgoVisitor) Visit(n ast.Node) Visitor {
	v.visitor = v.visitor.Visit(n)
	if v.visitor == nil {
		return nil
	}

	// For non-pgo nodes, let ast.Walk handle recursion.
	if _, isPgoNode := n.(Node); !isPgoNode {
		return v
	}

	switch n := n.(type) {
	case *Expr:
		Walk(v, n.Expr)

	case *StmtList:
		for _, stmt := range n.List {
			Walk(v, stmt)
		}

	case *FuncDecl:
		Walk(v, n.FuncDecl)

	case *GenDecl:
		Walk(v, n.GenDecl)

	case *Dots:
		// no children to visit

	case *AngleDots:
		for _, x := range n.List {
			Walk(v, x)
		}

	default:
		panic(fmt.Sprintf("pgoVisitor.Visit called with unknown node type %T", n))
	}

	return nil
}
