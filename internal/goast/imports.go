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
