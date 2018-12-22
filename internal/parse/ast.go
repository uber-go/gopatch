package parse

import (
	"go/ast"
	"go/token"
)

// Program is a single gopatch program consisting of one or more changes.
type Program struct {
	Changes []*Change
}

// Change is a single change in a program. Changes are specified in the
// format,
//
//  @@
//  # metavariables go here
//  @@
//  # diff goes here
//
// Optionally, a name may be specified for a change between the first two
// "@@"s.
//
//  @ mychange @
//  # metavariables go here
//  @@
//  # patch goes here
type Change struct {
	// Name for the change, if any.
	//
	// Names must be valid Go identifiers.
	Name string

	// Metavariables defined for this change.
	Meta *Meta

	// TODO: Diff
}

// Meta represents the metavariables section of a change.
//
// This consists of one or more declarations used in the patch.
type Meta struct {
	// Variables declared in this section.
	Vars []*VarDecl
}

// VarDecl is a single var declaration in a metavariable block.
//
//  var foo, bar identifier
//  var baz, qux expression
type VarDecl struct {
	// Position at which the "var" keyword appears.
	VarPos token.Pos

	// We're re-using the "go/ast".Ident type to represent identifiers in the
	// code so that we can track positional data for when identifiers appear
	// in the patch file.

	// Names of the variables declared in this statement.
	Names []*ast.Ident

	// Type of the variables.
	Type *ast.Ident
}

var _ ast.Node = (*VarDecl)(nil)

// Pos returns the position at which this declaration starts.
func (d *VarDecl) Pos() token.Pos { return d.VarPos }

// End returns the position of the next character after this declaration.
func (d *VarDecl) End() token.Pos {
	if d.Type != nil {
		return d.Type.End()
	}
	return token.NoPos
}
