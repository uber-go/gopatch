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

	// Comments for this change
	Comments []string
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
