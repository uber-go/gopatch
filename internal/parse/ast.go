package parse

import (
	"go/ast"
	"go/token"

	"github.com/uber-go/gopatch/internal/pgo"
)

// Program is a single gopatch program consisting of one or more changes.
type Program struct {
	Changes []*Change
}

// Change is a single change in a patch. Changes are specified in the format,
//
//  @@
//  # metavariables go here
//  @@
//  # patch goes here
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

	// Patch for this change.
	Patch *Patch
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

// Patch is the patch portion of the change containing the unified diff of the
// match/transformation.
type Patch struct {
	// Positions at which the entire patch begins and ends.
	StartPos, EndPos token.Pos

	// The before and after versions of the Patch broken apart from the
	// unified diff.
	Minus, Plus *pgo.File
}

var _ ast.Node = (*Patch)(nil)

// Pos returns the position at which this patch begins.
func (p *Patch) Pos() token.Pos { return p.StartPos }

// End returns the position immediately after this patch.
func (p *Patch) End() token.Pos { return p.EndPos }
