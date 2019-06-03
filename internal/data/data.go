package data

import (
	"fmt"
	"reflect"
)

// Data stores arbitrary data for Matchers and Replacers as they traverse or
// build the Go AST.
//
// Data provides an interface very similar to context.Context.
type Data interface {
	// Keys returns all known keys for this Data object in an unspecified
	// order.
	Keys() []interface{}

	// Values retrives the value associated with the given key or nil.
	Value(k interface{}) interface{}
}

// New builds a new empty Data.
func New() Data {
	return &emptyData{}
}

type emptyData struct{}

func (d *emptyData) Keys() []interface{}           { return nil }
func (d *emptyData) Value(interface{}) interface{} { return nil }

// WithValue returns a new Data with the given key-value pair associated with
// it.
//
//   d = data.WithValue(d, x, 42)
//   d.Value(x) // == 42
//
// The original object is left unmodified so if the returned Data object is
// discarded, its value will not be made available.
//
// Retreiving the value from this Data object is a linear time operation. To
// optimize for read-heavy use cases, use the Index function.
//
// Panics if either the key or the value are nil.
func WithValue(d Data, k, v interface{}) Data {
	if k == nil || v == nil {
		panic("key or value may not be nil")
	}
	return &valueData{Data: d, k: k, v: v}
}

type valueData struct {
	Data

	k, v interface{}
}

func (d *valueData) Keys() []interface{} {
	return append(d.Data.Keys(), d.k)
}

func (d *valueData) Value(k interface{}) interface{} {
	if k == d.k {
		return d.v
	}
	return d.Data.Value(k)
}

// Lookup retrieves the value associated with the given key and stores it into
// the pointer that vptr points to.
//
// Reports whether a value was found.
//
//   d = data.WithValue(d, x, 42)
//
//   var out int
//   data.Lookup(k, x, &out) // == true
//
// Panics if the type of the value for the pointer is not compatible with the
// value associated with the key.
func Lookup(d Data, k, vptr interface{}) (ok bool) {
	dest := reflect.ValueOf(vptr)
	if t := dest.Type(); t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Lookup target must be a pointer, not %v", t))
	}

	v := d.Value(k)
	if v != nil {
		dest.Elem().Set(reflect.ValueOf(v))
	}

	return v != nil
}

// Index returns a copy of the provided Data object where items are indexed
// for fast lookup.
//
// Lookups with Value or the Lookup function are constant time in the returned
// data object.
//
// Use this if your workload is divided between write-heavy and read-heavy
// portions.
func Index(d Data) Data {
	// Already indexed.
	if _, ok := d.(*indexedData); ok {
		return d
	}

	keys := d.Keys()
	items := make(map[interface{}]interface{})
	for _, k := range keys {
		items[k] = d.Value(k)
	}

	return &indexedData{
		items: items,
		keys:  keys,
	}
}

type indexedData struct {
	keys  []interface{}
	items map[interface{}]interface{}
}

func (d *indexedData) Keys() []interface{} {
	keys := make([]interface{}, len(d.keys))
	copy(keys, d.keys)
	return keys
}

func (d *indexedData) Value(k interface{}) interface{} {
	return d.items[k]
}
