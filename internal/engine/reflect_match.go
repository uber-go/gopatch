package engine

import (
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
)

// compileGeneric compiles a Matcher for arbitrary values inside a Go AST.
func (c *matcherCompiler) compileGeneric(v reflect.Value) (m Matcher) {
	switch v.Kind() {
	case reflect.Ptr:
		return c.compilePtr(v)
	case reflect.Slice:
		return c.compileSlice(v)
	case reflect.Struct:
		return c.compileStruct(v)
	case reflect.Interface:
		return c.compileInterface(v)
	default:
		return ValueMatcher{Type: v.Type(), Value: v.Interface()}
	}
}

// PtrMatcher matches a non-nil pointer in the AST.
type PtrMatcher struct {
	Matcher // underlying matcher
}

func (c *matcherCompiler) compilePtr(v reflect.Value) Matcher {
	// If the value is nil, we don't need to build the PtrMatcher.
	if v.IsNil() {
		return nilMatcher
	}

	return PtrMatcher{Matcher: c.compile(v.Elem())}
}

// Match matches a non-nil pointer.
func (m PtrMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	if got.Kind() != reflect.Ptr || got.IsNil() {
		return d, false
	}
	return m.Matcher.Match(got.Elem(), d)
}

// SliceMatcher matches a slice of values exactly.
//
// This matcher does not support eliding values with "...".
type SliceMatcher struct {
	// Matchers for individual fields of the slice.
	Items []Matcher
}

func (c *matcherCompiler) compileSlice(v reflect.Value) Matcher {
	if v.IsNil() {
		return nilMatcher
	}
	matchers := make([]Matcher, v.Len())
	for i := 0; i < v.Len(); i++ {
		matchers[i] = c.compile(v.Index(i))
	}
	return SliceMatcher{Items: matchers}
}

// Match mathces a slice of values.
func (m SliceMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	if got.Kind() != reflect.Slice || len(m.Items) != got.Len() {
		return d, false
	}

	for i, im := range m.Items {
		var ok bool
		d, ok = im.Match(got.Index(i), d)
		if !ok {
			return d, false
		}
	}

	return d, true
}

// StructMatcher matches a struct.
type StructMatcher struct {
	Type reflect.Type // type of the struct

	// Matchers for individual fields of the struct.
	Fields []Matcher
}

func (c *matcherCompiler) compileStruct(v reflect.Value) Matcher {
	typ := v.Type()
	fields := make([]Matcher, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		fields[i] = c.compile(v.Field(i))
	}

	return StructMatcher{
		Type:   typ,
		Fields: fields,
	}
}

// Match matches a struct.
func (m StructMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	if m.Type != got.Type() {
		return d, false
	}
	for i, f := range m.Fields {
		var ok bool
		d, ok = f.Match(got.Field(i), d)
		if !ok {
			return d, false
		}
	}
	return d, true
}

// InterfaceMatcher matches an interface value.
type InterfaceMatcher struct {
	Matcher // underlying matcher
}

func (c *matcherCompiler) compileInterface(v reflect.Value) Matcher {
	if v.IsNil() {
		return nilMatcher
	}
	return InterfaceMatcher{Matcher: c.compile(v.Elem())}
}

// Match matches non-nil interface nalues.
func (m InterfaceMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	if got.Kind() != reflect.Interface || got.IsNil() {
		return d, false
	}
	return m.Matcher.Match(got.Elem(), d)
}

// ValueMatcher matches a value as-is.
type ValueMatcher struct {
	Type  reflect.Type // underlying type
	Value interface{}  // value to match
}

// Match matches a value as-is.
func (m ValueMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	if m.Type != got.Type() {
		return d, false
	}
	return d, m.Value == got.Interface()
}
