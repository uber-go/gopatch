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

package parse

import (
	"fmt"
	"go/token"

	"github.com/uber-go/gopatch/internal/parse/section"
)

// Parse parses a Program.
func Parse(fset *token.FileSet, filename string, contents []byte) (*Program, error) {
	return newParser(fset).parseProgram(filename, contents)
}

type parser struct {
	fset *token.FileSet
}

func newParser(fset *token.FileSet) *parser {
	return &parser{fset: fset}
}

func (p *parser) errf(pos token.Pos, msg string, args ...any) error {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return fmt.Errorf("%v: %v", p.fset.Position(pos), msg)
}

func (p *parser) parseProgram(filename string, contents []byte) (*Program, error) {
	changes, err := section.Split(p.fset, filename, contents)
	if err != nil {
		return nil, err
	}

	prog := Program{Changes: make([]*Change, len(changes))}
	for i, c := range changes {
		change, err := p.parseChange(i, c)
		if err != nil {
			return nil, err
		}
		prog.Changes[i] = change
	}

	return &prog, nil
}
