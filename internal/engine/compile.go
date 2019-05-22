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
func (c *compiler) errf(pos token.Pos, msg string, args ...interface{}) {
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
