package main

import (
	"fmt"

	"conversion/to.git"
)

func foo(s string) *bool {
	fmt.Println("Hello World")
	if to.StrPtr(s) == nil {
		return to.BoolPtr(false)
	}

	if to.DecimalPtr(s) == nil {
		return to.BoolPtr(false)
	}

	return to.BoolPtr(true)
}
