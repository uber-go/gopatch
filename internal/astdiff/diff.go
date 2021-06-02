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

// Package astdiff provides a means of diffing Go AST nodes that may be
// mutated in-place by taking snapshots of their state between successive
// patch executions.
//
//  snap := astdiff.Before(node)
//  modify(node)
//  snap = snap.Diff(node, changelog)
//  modify(node)
//  snap = snap.Diff(node, changelog)
package astdiff

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/diff"
	"github.com/uber-go/gopatch/internal/goast"
)

// Changelog records the ranges of positions changed over the course
// of successive patches.
type Changelog interface {
	// Changed informs the changelog that the code in the range [pos, end)
	// has been modified.
	Changed(pos, end token.Pos)
}

// Before starts an AST diff.
//
// Given the unchanged node, it creates a Snapshot of the initial state before
// applying any patches.
func Before(n ast.Node, comments ast.CommentMap) *Snapshot {
	return &Snapshot{snapshot(reflect.ValueOf(n), comments)}
}

// Snapshot maintains the current view of the ast.Node being altered over the
// course of patching.
type Snapshot struct {
	value *value
}

// Diff compares the current Snapshot to the altered Node, updates the provided
// Changelog, and returns an updated Snapshot.
func (s *Snapshot) Diff(n ast.Node, cl Changelog) *Snapshot {
	f := changeFinder{
		Region: Region{
			Pos: s.value.Pos(),
			End: s.value.End(),
		},
		cl: cl,
	}

	v := snapshot(reflect.ValueOf(n).Convert(s.value.Type()), nil)
	f.Walk(s.value, v)
	return &Snapshot{value: v}
}

// Region denotes the consecutive positions
type Region struct{ Pos, End token.Pos }

type changeFinder struct {
	Region

	cl Changelog
}

func (f changeFinder) unchanged(from, to *value) {
	to.Comments = from.Comments
}

func (f changeFinder) changed() {
	f.cl.Changed(f.Pos, f.End)

}

func (f changeFinder) commentsFor(n *value) (before, after []*ast.Comment) {
	pos, end := n.Pos(), n.End()
	for _, cg := range n.Comments {
		if cg.End() <= pos {
			before = append(before, cg.List...)
		}

		if cg.Pos() >= end {
			after = append(after, cg.List...)
		}
	}

	return before, after
}

func (f changeFinder) Walk(from, to *value) (equal bool) {
	if from.Type() != to.Type() {
		// Can't traverse into this subtree.
		f.changed()
		return false
	}

	switch from.Type() {
	case goast.ObjectPtrType:
		// Ident.obj forms a cycle.
		return true
	case goast.CommentGroupPtrType:
		return true
	case goast.PosType:
		fromPos := from.Interface().(token.Pos)
		toPos := to.Interface().(token.Pos)
		if fromPos.IsValid() != toPos.IsValid() {
			f.changed()
			return false
		}
		f.unchanged(from, to)
		return true
	}

	switch from.Kind() {
	case reflect.Ptr, reflect.Interface:
		// If LHS didn't exist, whether RHS exists or not doesn't affect the
		// Changed sections.
		if from.IsNil() {
			return
		}

		// If LHS was deleted, this whole section has changed.
		if to.IsNil() {
			f.changed()
			return false
		}

		// Dereferencing a pointer or interface doesn't affect region.
		if f.Walk(from.Elem, to.Elem) {
			f.unchanged(from, to)
			return true
		}
		return false

	case reflect.Slice:
		if f.walkSlice(from, to) {
			f.unchanged(from, to)
			return true
		}
		return false

	case reflect.Struct:
		if f.walkStruct(from, to) {
			f.unchanged(from, to)
			return true
		}
		return false

	default:
		if from.Interface() == to.Interface() {
			f.unchanged(from, to)
			return true
		}

		f.changed()
		return false
	}
}

func (f changeFinder) walkStruct(from, to *value) bool {
	// The order of fields in AST structs matches how the elements appear in
	// the code so we can treat fields as siblings

	starts := make([]token.Pos, from.Len())
	lastEnd := f.Pos
	for i, f := range from.Children {
		switch {
		case f.IsNode:
			// If the field is a Node, its range begins when the Node starts.
			starts[i] = f.Pos()
			lastEnd = f.End()
		case f.Type() == goast.PosType:
			// If the field is a token.Pos, its range begins based on whatever
			// its value is.
			pos := f.Interface().(token.Pos)
			if pos.IsValid() {
				starts[i] = pos
			}
		default:
			// Otherwise the start position is the end position of the last
			// valid node.
			starts[i] = lastEnd
		}
	}

	ends := make([]token.Pos, from.Len())
	nextPos := f.End
	for i := from.Len() - 1; i >= 0; i-- {
		ends[i] = nextPos
		if v := from.Children[i]; v.IsNode {
			// If the field is a Node, its range ends where the Node ends.
			ends[i] = v.End()
		}

		// The field that preceds this field should use this field's start
		// position as its end position if it's not a Node.
		nextPos = starts[i]
	}

	equal := true
	for i, v := range from.Children {
		f.Pos = starts[i]
		f.End = ends[i]
		equal = f.Walk(v, to.Children[i]) && equal
	}
	return equal
}

func (f changeFinder) walkSlice(from, to *value) bool {
	// If this isn't a slice of nodes, require that they're equal in length
	// and check all elements.
	if !from.Type().Elem().Implements(goast.NodeType) {
		if from.Len() != to.Len() {
			f.changed()
			return false
		}

		equal := true
		for i := 0; i < from.Len(); i++ {
			equal = f.Walk(from.Children[i], to.Children[i]) && equal
		}
		return equal
	}

	es := diff.Difference(from.Len(), to.Len(), func(i, j int) diff.Result {
		return compareNodes(from.Children[i], to.Children[j])
	})

	regions := make([]Region, from.Len())
	for i, n := range from.Children {
		r := f.Region

		// Extend to sibling beinning and end.
		if i > 0 {
			r.Pos = from.Children[i-1].End()

			// If the previous node has any trailing comments, maintain
			// whitespace between them and us.
			if _, after := f.commentsFor(from.Children[i-1]); len(after) > 0 {
				r.Pos = n.Pos()
			}
		}
		if i < from.Len()-1 {
			r.End = from.Children[i+1].Pos()

			// If the next node has leading comments, maintain whitespace
			// between them and us.
			if before, _ := f.commentsFor(from.Children[i+1]); len(before) > 0 {
				r.End = n.End()
			}
		}

		// If we have any comments associated with us, clamp to them.
		before, after := f.commentsFor(n)
		if len(before) > 0 {
			r.Pos = maxPos(r.Pos, before[len(before)-1].End())
		}

		if len(after) > 0 {
			r.End = minPos(r.End, after[0].Pos())
		}

		regions[i] = r
	}

	equal := true
	var i, j int
	for _, e := range es {
		switch e {
		case diff.Identity:
			f.unchanged(from.Children[i], to.Children[j])
			i++
			j++

		case diff.Modified:
			equal = false
			f.Region = regions[i]
			f.Walk(from.Children[i], to.Children[j])
			i++
			j++

		case diff.UniqueX:
			equal = false
			f.Region = regions[i]
			f.changed() // deleted
			i++

		case diff.UniqueY:
			equal = false // new item, nothing to do
			j++
		}
	}

	return equal
}

type nodeComparer struct{ diff.Result }

func compareNodes(from, to *value) diff.Result {
	var c nodeComparer
	c.Walk(from, to)
	return c.Result
}

func (c *nodeComparer) Walk(from, to *value) {
	if from.Type() != to.Type() {
		c.NumDiff += 2 // not equal or similar
		return
	}

	switch from.Type() {
	case goast.ObjectPtrType:
		// Ident.obj forms a cycle.
		return
	case goast.PosType:
		fromPos := from.Interface().(token.Pos)
		toPos := to.Interface().(token.Pos)
		if fromPos.IsValid() != toPos.IsValid() {
			c.NumDiff++
		} else {
			c.NumSame++
		}
		return
	}

	switch from.Kind() {
	case reflect.Ptr, reflect.Interface:
		if from.IsNil() || to.IsNil() {
			if from.IsNil() == to.IsNil() {
				c.NumSame++
			} else {
				c.NumDiff++
			}
		} else {
			c.Walk(from.Elem, to.Elem)
		}

	case reflect.Slice:
		results := make([][]diff.Result, from.Len())
		for i := range results {
			results[i] = make([]diff.Result, to.Len())
		}

		es := diff.Difference(from.Len(), to.Len(), func(i, j int) diff.Result {
			result := compareNodes(from.Children[i], to.Children[j])
			results[i][j] = result
			return result
		})

		var i, j int
		for _, e := range es {
			switch e {
			case diff.Identity, diff.Modified:
				result := results[i][j]
				c.NumDiff += result.NumDiff
				c.NumSame += result.NumSame
				i++
				j++
			case diff.UniqueX:
				i++
				c.NumDiff++
			case diff.UniqueY:
				j++
				c.NumDiff++
			}
		}

	case reflect.Struct:
		for i, v := range from.Children {
			c.Walk(v, to.Children[i])
		}

	default:
		if from.Interface() == to.Interface() {
			c.NumSame++
		} else {
			c.NumDiff++
		}
	}
}
