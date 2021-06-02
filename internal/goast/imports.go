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
	"fmt"
	"go/ast"
	"strconv"
)

// ImportPath returns the import path of the provided Go import.
func ImportPath(spec *ast.ImportSpec) string {
	if spec == nil || spec.Path == nil {
		panic("ImportSpec and its Path must be non-nil")
	}

	path, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		panic(fmt.Sprintf("invalid import path %q: %v", spec.Path.Value, err))
	}
	return path
}

// ImportName returns the name of the provided import or an empty string if
// it's not a named import.
func ImportName(spec *ast.ImportSpec) string {
	if spec == nil {
		panic("ImportSpec must be non-nil")
	}

	var name string
	if spec.Name != nil {
		name = spec.Name.Name
	}
	return name
}

// FindImportSpec finds and returns the ImportSpec for an import with the
// given import path, returning nil if a matching ImportSpec was not found.
func FindImportSpec(f *ast.File, path string) *ast.ImportSpec {
	if f == nil {
		panic("File must be non-nil")
	}

	for _, got := range f.Imports {
		if ImportPath(got) == path {
			return got
		}
	}
	return nil
}
