package engine

import "reflect"

func refl(v interface{}) reflect.Value {
	return reflect.ValueOf(v)
}

func stringPtr(s string) *string { return &s }
