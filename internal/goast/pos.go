package goast

import (
	"go/token"
	"reflect"
)

type transformOptions struct {
	SkipTypes []reflect.Type
}

// TransformOption customizes the behavior of TransformPos.
type TransformOption func(*transformOptions)

// SkipTypes specifies types which shold not be transformed by TransformPos.
func SkipTypes(types ...reflect.Type) TransformOption {
	return func(opts *transformOptions) {
		opts.SkipTypes = append(opts.SkipTypes, types...)
	}
}

// TransformPos runs the given function on all token.Pos values found in the
// given object and its descendants, replacing them in-place with the value
// produced by the provided function.
//
// Caveat: Free-floating comments on File objects are not handled by
// TransformPos.
func TransformPos(n interface{}, transform func(token.Pos) token.Pos, opts ...TransformOption) {
	v, ok := n.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(n)
	}

	var options transformOptions
	for _, opt := range opts {
		opt(&options)
	}

	transformPos(v, transform, &options)
}

// OffsetPos offsets all token.Pos values found in the given object and its
// descendants in-place.
//
// Caveat: Free-floating comments on File objects are not handled by
// OffsetPos.
func OffsetPos(n interface{}, offset int, opts ...TransformOption) {
	TransformPos(n, func(pos token.Pos) token.Pos {
		return pos + token.Pos(offset)
	}, opts...)
}

func transformPos(v reflect.Value, transformFn func(token.Pos) token.Pos, opts *transformOptions) {
	if !v.IsValid() {
		return
	}

	vt := v.Type()
	for _, t := range opts.SkipTypes {
		if t == vt {
			return
		}
	}

	switch vt {
	case FilePtrType:
		if v.IsNil() {
			return
		}
		v = v.Elem()

		// ast.File maintains a bunch of internal references. Only the
		// following fields are unique references.
		transformPos(v.FieldByName("Doc"), transformFn, opts)
		transformPos(v.FieldByName("Package"), transformFn, opts)
		transformPos(v.FieldByName("Name"), transformFn, opts)
		transformPos(v.FieldByName("Decls"), transformFn, opts)

		// NOTE: File.Comments contains both, comments that document
		// objects (also referenced by those nodes' Doc fields), and
		// free-floating comments in the file. Rather than tracking
		// whether a comment has already been processed, we're just
		// not going to handle free-floating comments until it becomes
		// necessary.
	case ObjectPtrType:
		// ast.Object has a reference to the target object, causing
		// cyclic references. Since the underlying object isn't
		// changing, we don't need to do anything here.
	case PosType:
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
				transformPos(v.Index(i), transformFn, opts)
			}
		case reflect.Interface, reflect.Ptr:
			transformPos(v.Elem(), transformFn, opts)
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				transformPos(v.Field(i), transformFn, opts)
			}
		case reflect.Map:
			// go/ast does not use maps in the AST besides Scope objects
			// which are attached to File objects, which we handle
			// explicitly.
			panic("cannot use maps inside an AST node")
		}
	}
}
