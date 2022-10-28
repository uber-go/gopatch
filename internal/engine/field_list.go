// Copyright (c) 2022 Uber Technologies, Inc.
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
	"errors"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
)

type OptionalFieldListMatcher struct {
	Dots token.Pos
}

func (c *matcherCompiler) compileFieldList(v reflect.Value) Matcher {
	flist := v.Interface().(*ast.FieldList)

	// Special-case: (...) matches lack of parens.
	// (foo, ..., bar) matches like a regular field list.
	var (
		isDots bool
		dotPos token.Pos
	)
	if flist != nil && len(flist.List) == 1 {
		_, isDots = flist.List[0].Type.(*pgo.Dots)
		if isDots {
			dotPos = flist.List[0].Pos()
		}
	}
	if !isDots {
		return c.compileGeneric(v)
	}

	c.dots = append(c.dots, dotPos)
	return OptionalFieldListMatcher{Dots: dotPos}
}

func (m OptionalFieldListMatcher) Match(got reflect.Value, d data.Data, r Region) (data.Data, bool) {
	var fields []forDotsField

	got = got.Elem()
	t := got.Type()
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, forDotsField{Idx: i, Value: got.Field(i)})
	}

	d = data.WithValue(
		d,
		fieldListKey(m.Dots),
		fieldListData{
			Fields: fields,
			Region: r,
		},
	)
	return d, true
}

type fieldListKey int

type fieldListData struct {
	Fields []forDotsField
	Region Region
}

type OptionalFieldListReplacer struct {
	Dots     token.Pos
	dotAssoc map[token.Pos]token.Pos
}

func (c *replacerCompiler) compileFieldList(v reflect.Value) Replacer {
	flist := v.Interface().(*ast.FieldList)
	var (
		isDots bool
		dotPos token.Pos
	)
	if fields := flist.List; len(fields) == 1 {
		_, isDots = fields[0].Type.(*pgo.Dots)
		if isDots {
			dotPos = fields[0].Pos()
		}
	}
	if !isDots {
		return c.compileGeneric(v)
	}

	c.dots = append(c.dots, dotPos)
	return OptionalFieldListReplacer{
		Dots:     dotPos,
		dotAssoc: c.dotAssoc,
	}
}

func (r OptionalFieldListReplacer) Replace(d data.Data, cl Changelog, pos token.Pos) (reflect.Value, error) {
	var fd fieldListData
	if !data.Lookup(d, fieldListKey(r.dotAssoc[r.Dots]), &fd) {
		return reflect.Value{}, errors.New("match data not found for '(...)': " +
			"are you sure that the line appears on a context line without a preceding '-' or '+'?")
	}
	cl.Unchanged(fd.Region.Pos, fd.Region.End)

	flist := reflect.New(goast.FieldListType).Elem()
	for _, f := range fd.Fields {
		flist.Field(f.Idx).Set(f.Value)
	}

	return flist.Addr(), nil
}
