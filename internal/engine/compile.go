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
	"errors"
	"fmt"
	"go/token"

	"github.com/uber-go/gopatch/internal/parse"
	"go.uber.org/multierr"
)

// Program is a collection of compiled changes.
type Program struct {
	Changes []*Change
}

// Compile compiles a parsed gopatch Program.
func Compile(fset *token.FileSet, p *parse.Program) (*Program, error) {
	c := newCompiler(fset)
	return c.compileProgram(p), c.Err()
}

type compiler struct {
	fset   *token.FileSet
	errors []error
}

func newCompiler(fset *token.FileSet) *compiler {
	return &compiler{fset: fset}
}

// Convenience function to build error messages with positioning data.
func (c *compiler) errf(pos token.Pos, msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	if pos.IsValid() {
		msg = fmt.Sprintf("%v: %v", c.fset.Position(pos), msg)
	}
	c.errors = append(c.errors, errors.New(msg))
}

// Err collates all the errors encountered during compilation and returns
// them.
func (c *compiler) Err() error {
	return multierr.Combine(c.errors...)
}

// Compiles a Program.
func (c *compiler) compileProgram(aprogram *parse.Program) *Program {
	var p Program
	for _, achange := range aprogram.Changes {
		if change := c.compileChange(achange); change != nil {
			p.Changes = append(p.Changes, change)
		}
	}
	return &p
}
