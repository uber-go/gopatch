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

package engine

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
)

// MetavarMatcher is compiled from a metavarible occurring in the minus
// section of the patch.
//
//  @@
//  var x expression
//  @@
//  -foo(x, x)
//
// The first time a metavariable occurs in the patch, it matches and captures
// a value of the requested type. For any consecutive appearances of the
// metavariable, the previously captured value is expected to match.
//
// For example, the patch above will match any expression for the first "x"
// and the second occurrence will require the previously captured expression
// to match.
type MetavarMatcher struct {
	Fset *token.FileSet

	// Name of the metavariable.
	Name string

	// Reports whether the provided type matches the metavariable declaration.
	TypeMatches func(reflect.Type) bool
}

func (c *matcherCompiler) compileIdent(v reflect.Value) Matcher {
	ident := v.Interface().(*ast.Ident)
	if ident == nil {
		return nilMatcher
	}

	name := ident.Name
	var matchType func(reflect.Type) bool
	switch c.meta.LookupVar(name) {
	case ExprMetavarType:
		matchType = isExpression
	case IdentMetavarType:
		matchType = isIdent
	default:
		// Not a metavariable. Match the identifer as-is.
		return c.compileGeneric(v)
	}

	return MetavarMatcher{
		Fset:        c.fset,
		Name:        name,
		TypeMatches: matchType,
	}
}

// Match matches a metavariable from the patch in the AST.
func (m MetavarMatcher) Match(got reflect.Value, d data.Data, r Region) (data.Data, bool) {
	if !m.TypeMatches(got.Type()) {
		return d, false
	}

	key := metavarKey(m.Name)

	var md metavarData
	if data.Lookup(d, key, &md) {
		// We've already seen this metavariable. Match the value without
		// altering captured data.
		_, ok := md.Match(got, data.New(), r)
		return d, ok
	}

	// We're seeing this for the first time. Capture it into a compiler and
	// replacer so we can match and reproduce it later.
	return data.WithValue(d, key, metavarData{
		Matcher:  newMatcherCompiler(m.Fset, nil, r.Pos, r.End).compile(got),
		Replacer: newReplacerCompiler(m.Fset, nil, r.Pos, r.End).compile(got),
	}), true
}

type metavarKey string

type metavarData struct {
	Matcher
	Replacer
}

func isExpression(t reflect.Type) bool {
	return t.Implements(goast.ExprType)
}

func isIdent(t reflect.Type) bool {
	return t == goast.IdentPtrType
}

// MetavarReplacer is compiled from a metavarible occurring in the plus
// section of the patch.
//
//  @@
//  var x expression
//  @@
//  -foo(x)
//  +bar(x)
//
// A metavariable cannot be referenced in the plus secton of the patch if it
// wasn't in the minus section. For example, the following is invalid.
//
//  @@
//  var x expression
//  @@
//  -foo()
//  +foo(x)
//
// Each occurrence of the metavariable in the plus section of the patch is
// replaced with the value originally captured for it by the Matcher.
type MetavarReplacer struct {
	Name string
}

func (c *replacerCompiler) compileIdent(v reflect.Value) Replacer {
	name := v.Interface().(*ast.Ident).Name
	if c.meta.LookupVar(name) == 0 {
		// Not a metavariable. Reproduce the identifier as-is.
		return c.compileGeneric(v)
	}
	return MetavarReplacer{Name: name}
}

// Replace reproduces the value of a matched metavariable.
func (m MetavarReplacer) Replace(d data.Data, cl Changelog, pos token.Pos) (reflect.Value, error) {
	key := metavarKey(m.Name)

	var md metavarData
	if !data.Lookup(d, key, &md) {
		// This will happen only if a metavariable was referenced in the plus
		// section without being referenced in the minus section.
		// TODO(abg): Guard against that during compilation instead.
		return reflect.Value{}, fmt.Errorf("could not find value for metavariable %q", m.Name)
	}

	return md.Replace(data.New(), cl, pos)
}
