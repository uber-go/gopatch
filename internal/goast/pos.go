package goast

import (
	"go/ast"
	"go/token"
	"reflect"
)

// TransformPos runs the given function on all token.Pos values found in the
// given object and its descendants, replacing them in-place with the value
// produced by the provided function.
func TransformPos(n interface{}, transform func(token.Pos) token.Pos) {
	transformPos(reflect.ValueOf(n), transform)
}

// OffsetPos offsets all token.Pos values found in the given object and its
// descendants in-place.
func OffsetPos(n interface{}, offset int) {
	TransformPos(n, func(pos token.Pos) token.Pos { return pos + token.Pos(offset) })
}

var (
	posType    = reflect.TypeOf(token.Pos(0))
	objectType = reflect.TypeOf((*ast.Object)(nil))
)

func transformPos(v reflect.Value, transformFn func(token.Pos) token.Pos) {
	if !v.IsValid() {
		return
	}

	// token.Pos is present only as a struct field. Every other composite type
	// can be recursed.
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			transformPos(v.Index(i), transformFn)
		}
	case reflect.Interface, reflect.Ptr:
		transformPos(v.Elem(), transformFn)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			switch f.Type() {
			case objectType:
				// ast.Object has a reference to the target object, causing
				// cyclic references. Since the underlying object isn't
				// changing, we don't need to do anything here.
			case posType:
				pos := token.Pos(f.Int())
				if pos.IsValid() {
					// We want to change Pos only if it's valid, as in
					// non-zero. There are parts in the Go AST where the
					// presence of a valid Pos changes the generated syntax
					// significantly. One example is type aliases.
					//
					//   type Foo = Bar
					//   type Foo Bar
					//
					// The only difference between the parsed representations
					// for the two type declarations above is whether the
					// Equals field has a valid token.Pos or not. If the Pos
					// is invalid, we don't want to change it.
					f.SetInt(int64(transformFn(pos)))
				}
			default:
				transformPos(f, transformFn)
			}
		}
	case reflect.Map:
		// go/ast does not use maps in the AST. Neither do we.
		panic("cannot use maps inside an AST node")
	}
}
