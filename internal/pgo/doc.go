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

// Package pgo defines a superset of the Go syntax and provides the ability to
// parse and manipulate it. pgo is used to parse versions of a gopatch change.
//
// Key features of pgo are:
//   - package names are optional
//   - expressions and statements may be written at the top level
//   - ... is supported in a number of places
//
// # Syntax
//
// The following documents the syntax for pgo, assuming the syntax for Go is
// already provided under the "go." namespace.
//
//	// Package names and imports are optional. Only a single top-level
//	// declaration is allowed.
//	file = package? imports decl;
//	package = "package" go.package_name;
//
//	// import_decl is a standard Go import declarations.
//	imports = go.import_decl*;
//
//	// For top-level declarations, type, function/method, and constant
//	// declarations are assumed to be standard Go declarations. In addition to
//	// them, statement lists and expressions are supported at the top-level.
//	decl
//	  = go.type_decl
//	  | go.func_decl
//	  | go.const_decl
//	  | stmt_list
//	  | go.expr;
//
//	// Statement lists can open with curly braces or as-is. The two cases are
//	// necessary to allow disambiguating between top-level type/const
//	// declarations and those inlined inside a code block.
//	stmt_list = '{' go.stmt* '}' | go.stmt+;
//
// # Dots
//
// "...", referred to as "dots" is accepted anywhere as a statement and an
// expression. Note that pgo doesn't ascribe any meaning to these dots but
// gopatch may.
package pgo
