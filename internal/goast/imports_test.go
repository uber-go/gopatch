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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportPath(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Panics(t, func() {
			ImportPath(nil)
		})
	})

	t.Run("nil path", func(t *testing.T) {
		assert.Panics(t, func() {
			ImportPath(&ast.ImportSpec{})
		})
	})

	t.Run("undecodable import path", func(t *testing.T) {
		assert.Panics(t, func() {
			ImportPath(&ast.ImportSpec{
				Path: &ast.BasicLit{Value: "foo"},
			})
		})
	})

	t.Run("success", func(t *testing.T) {
		got := ImportPath(&ast.ImportSpec{Path: &ast.BasicLit{Value: `"foo"`}})
		assert.Equal(t, "foo", got)
	})
}

func TestImportName(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Panics(t, func() {
			ImportName(nil)
		})
	})

	t.Run("unnamed import", func(t *testing.T) {
		got := ImportName(&ast.ImportSpec{
			Path: &ast.BasicLit{Value: `"foo"`},
		})
		assert.Empty(t, got)
	})

	t.Run("named import", func(t *testing.T) {
		got := ImportName(&ast.ImportSpec{
			Name: ast.NewIdent("bar"),
			Path: &ast.BasicLit{Value: `"foo"`},
		})
		assert.Equal(t, "bar", got)
	})
}

func TestFindImportSpec(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Panics(t, func() {
			FindImportSpec(nil, "foo")
		})
	})

	foo := &ast.ImportSpec{Path: &ast.BasicLit{Value: `"foo"`}}
	fooBar := &ast.ImportSpec{Path: &ast.BasicLit{Value: `"foo/bar"`}}
	file := &ast.File{
		Imports: []*ast.ImportSpec{foo, fooBar},
	}

	t.Run("no match", func(t *testing.T) {
		assert.Nil(t, FindImportSpec(file, "foo/bar/baz"))
	})

	t.Run("match foo", func(t *testing.T) {
		assert.Equal(t, foo, FindImportSpec(file, "foo"))
	})

	// not using / in test name because that's how subtests are separated.
	t.Run("match foo-bar", func(t *testing.T) {
		assert.Equal(t, fooBar, FindImportSpec(file, "foo/bar"))
	})
}
