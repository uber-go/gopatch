package engine

import "reflect"

// compileGeneric compiles a Replacer for arbitrary values inside a Go AST.
func (c *replacerCompiler) compileGeneric(v reflect.Value) (r Replacer) {
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
		return ValueReplacer{Value: v}
	}
}

// PtrReplacer replaces a pointer type.
type PtrReplacer struct {
	Replacer // underlying replacer

	Type reflect.Type // type of the pointer
}

func (c *replacerCompiler) compilePtr(v reflect.Value) Replacer {
	if v.IsNil() {
		return ZeroReplacer{Type: v.Type()}
	}

	return PtrReplacer{
		Type:     v.Type(),
		Replacer: c.compile(v.Elem()),
	}
}

// Replace replaces a pointer type.
func (r PtrReplacer) Replace() (reflect.Value, error) {
	x, err := r.Replacer.Replace()
	if err != nil {
		return reflect.Value{}, err
	}

	v := reflect.New(r.Type).Elem()
	v.Set(x.Addr())
	return v, nil
}

// SliceReplacer replaces a slice of values.
//
// This replacer does not support generating values elided with "...".
type SliceReplacer struct {
	Type reflect.Type // type of slice

	// Replacers for individual items in the slice.
	Items []Replacer
}

func (c *replacerCompiler) compileSlice(v reflect.Value) Replacer {
	if v.IsNil() {
		return ZeroReplacer{Type: v.Type()}
	}

	items := make([]Replacer, v.Len())
	for i := 0; i < v.Len(); i++ {
		items[i] = c.compile(v.Index(i))
	}

	return SliceReplacer{
		Type:  v.Type(),
		Items: items,
	}
}

// Replace replaces a slice.
func (r SliceReplacer) Replace() (reflect.Value, error) {
	v := reflect.MakeSlice(r.Type, len(r.Items), len(r.Items))
	for i, itemR := range r.Items {
		item, err := itemR.Replace()
		if err != nil {
			return reflect.Value{}, err
		}
		v.Index(i).Set(item)
	}

	return v, nil
}

// StructReplacer replaces a struct.
type StructReplacer struct {
	Type reflect.Type // type of the struct

	// Replacers for individual fields of the struct.
	Fields []Replacer
}

func (c *replacerCompiler) compileStruct(v reflect.Value) Replacer {
	typ := v.Type()

	fields := make([]Replacer, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		fields[i] = c.compile(v.Field(i))
	}

	return StructReplacer{
		Type:   typ,
		Fields: fields,
	}
}

// Replace replaces a struct value.
func (r StructReplacer) Replace() (reflect.Value, error) {
	v := reflect.New(r.Type).Elem()
	for i, f := range r.Fields {
		fv, err := f.Replace()
		if err != nil {
			return reflect.Value{}, err
		}
		v.Field(i).Set(fv)
	}
	return v, nil
}

// InterfaceReplacer replaces an interface value.
type InterfaceReplacer struct {
	Replacer // underlying replacer

	Type reflect.Type // type of the interface
}

func (c *replacerCompiler) compileInterface(v reflect.Value) Replacer {
	if v.IsNil() {
		return ZeroReplacer{Type: v.Type()}
	}
	return InterfaceReplacer{
		Type:     v.Type(),
		Replacer: c.compile(v.Elem()),
	}
}

// Replace replaces an interface value.
func (r InterfaceReplacer) Replace() (reflect.Value, error) {
	x, err := r.Replacer.Replace()
	if err != nil {
		return reflect.Value{}, err
	}

	v := reflect.New(r.Type).Elem()
	v.Set(x)
	return v, nil
}

// ValueReplacer replace a value as-is.
type ValueReplacer struct{ Value reflect.Value }

// Replace replaces a value as-is.
func (r ValueReplacer) Replace() (reflect.Value, error) {
	return r.Value, nil
}
