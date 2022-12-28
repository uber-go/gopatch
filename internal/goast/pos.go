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

// TransformPos runs the given function on all token.Pos values found in the
// given object and its descendants, replacing them in-place with the value
// produced by the provided function.
//
// Caveat: Free-floating comments on File objects are not handled by
// TransformPos.
func TransformPos(n any, transform func(token.Pos) token.Pos) {
	transformPos(reflect.ValueOf(n), transform)
}

// OffsetPos offsets all token.Pos values found in the given object and its
// descendants in-place.
//
// Caveat: Free-floating comments on File objects are not handled by
// OffsetPos.
func OffsetPos(n any, offset int) {
	TransformPos(n, func(pos token.Pos) token.Pos { return pos + token.Pos(offset) })
}

var (
	posType       = reflect.TypeOf(token.Pos(0))
	objectPtrType = reflect.TypeOf((*ast.Object)(nil))
	fileType      = reflect.TypeOf(ast.File{})
)

func transformPos(v reflect.Value, transformFn func(token.Pos) token.Pos) {
	if !v.IsValid() {
		return
	}

	switch v.Type() {
	case fileType:
		// ast.File maintains a bunch of internal references. Only the
		// following fields are unique references.
		transformPos(v.FieldByName("Doc"), transformFn)
		transformPos(v.FieldByName("Package"), transformFn)
		transformPos(v.FieldByName("Name"), transformFn)
		transformPos(v.FieldByName("Decls"), transformFn)

		// NOTE: File.Comments contains both, comments that document
		// objects (also referenced by those nodes' Doc fields), and
		// free-floating comments in the file. Rather than tracking
		// whether a comment has already been processed, we're just
		// not going to handle free-floating comments until it becomes
		// necessary.
	case objectPtrType:
		// ast.Object has a reference to the target object, causing
		// cyclic references. Since the underlying object isn't
		// changing, we don't need to do anything here.
	case posType:
		pos := token.Pos(v.Int())
		if pos.IsValid() {
			// We want to change Pos only if it's valid, as in
			// non-zero. There are parts in the Go AST where the
			// presence of a valid Pos changes the generated
			// syntax significantly. One example is type aliases.
			//
			//   type Foo = Bar
			//   type Foo Bar
			//
			// The only difference between the parsed
			// representations for the two type declarations above
			// is whether the Equals field has a valid token.Pos
			// or not. If the Pos is invalid, we don't want to
			// change it.
			v.SetInt(int64(transformFn(pos)))
		}
	default:
		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				transformPos(v.Index(i), transformFn)
			}
		case reflect.Interface, reflect.Ptr:
			transformPos(v.Elem(), transformFn)
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				transformPos(v.Field(i), transformFn)
			}
		case reflect.Map:
			// go/ast does not use maps in the AST besides Scope objects
			// which are attached to File objects, which we handle
			// explicitly.
			panic("cannot use maps inside an AST node")
		}
	}
}
