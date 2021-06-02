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

package astdiff

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/goast"
)

type value struct {
	t reflect.Type

	// Only one of the following three is set.
	isNil    bool
	value    interface{}
	Elem     *value
	Children []*value

	// Set only if this is a Node.
	IsNode   bool
	pos, end token.Pos
	Comments []*ast.CommentGroup
}

func (v *value) Type() reflect.Type     { return v.t }
func (v *value) Kind() reflect.Kind     { return v.t.Kind() }
func (v *value) Pos() token.Pos         { return v.pos }
func (v *value) End() token.Pos         { return v.end }
func (v *value) Interface() interface{} { return v.value }
func (v *value) Len() int               { return len(v.Children) }
func (v *value) IsNil() bool            { return v.isNil }

func snapshot(v reflect.Value, cmap ast.CommentMap) (val *value) {
	t := v.Type()

	switch t {
	case goast.ObjectPtrType:
		// Ident.obj forms a cycle.
		return &value{t: t, isNil: true}
	case goast.CommentGroupPtrType, goast.ScopePtrType:
		// Snapshots don't care about comment groups.
		return &value{t: t, isNil: true}
	case goast.PosType:
		return &value{
			t:     t,
			value: v.Interface(),
		}
	}

	if t.Implements(goast.NodeType) && !v.IsNil() {
		defer func(n ast.Node) {
			val.IsNode = true
			val.pos = n.Pos()
			val.end = n.End()
			val.Comments = cmap[n]
		}(v.Interface().(ast.Node))
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return &value{t: t, isNil: true}
		}

		return &value{
			t:    t,
			Elem: snapshot(v.Elem(), cmap),
		}
	case reflect.Slice:
		children := make([]*value, v.Len())
		for i := 0; i < v.Len(); i++ {
			children[i] = snapshot(v.Index(i), cmap)
		}
		return &value{
			t:        t,
			Children: children,
		}

	case reflect.Struct:
		children := make([]*value, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			children[i] = snapshot(v.Field(i), cmap)
		}
		return &value{
			t:        t,
			Children: children,
		}

	default:
		return &value{
			t:     t,
			value: v.Interface(),
		}
	}
}

func minPos(l, r token.Pos) token.Pos {
	if l < r {
		return l
	}
	return r
}

func maxPos(l, r token.Pos) token.Pos {
	if l > r {
		return l
	}
	return r
}
