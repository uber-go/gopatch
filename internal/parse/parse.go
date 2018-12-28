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

func (p *parser) errf(pos token.Pos, msg string, args ...interface{}) error {
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
