package main

import (
	"bufio"
	"fmt"
	"go/token"
	"io"
	"os"

	"github.com/uber-go/gopatch/internal/engine"
	"github.com/uber-go/gopatch/internal/parse"
	"go.uber.org/multierr"
)

// patchLoader loads patches from varying sources
// and compiles them into a series of programs.
type patchLoader struct {
	fset  *token.FileSet
	progs []*engine.Program

	// Pointer to parseAndCompile function,
	// which we can use to swap out this logic.
	parseAndCompile func(*token.FileSet, string, []byte) (*engine.Program, error)
}

func newPatchLoader(fset *token.FileSet) *patchLoader {
	return &patchLoader{
		fset:            fset,
		parseAndCompile: parseAndCompile,
	}
}

func (l *patchLoader) Programs() []*engine.Program {
	return l.progs
}

// LoadReader loads a patch from an io.Reader.
func (l *patchLoader) LoadReader(name string, r io.Reader) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	prog, err := l.parseAndCompile(l.fset, name, src)
	if err != nil {
		return err
	}

	l.progs = append(l.progs, prog)
	return nil
}

// LoadFile loads a patch from the given file.
func (l *patchLoader) LoadFile(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(f))

	if err := l.LoadReader(path, f); err != nil {
		return err
	}

	return nil
}

// LoadFileList loads patches specified in a file
// that contains a list of file paths to other patches.
func (l *patchLoader) LoadFileList(patchList string) (err error) {
	f, err := os.Open(patchList)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(f))

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		path := scanner.Text()
		if len(path) == 0 {
			continue
		}

		if err := l.LoadFile(path); err != nil {
			return fmt.Errorf("load patch %q: %w", path, err)
		}
	}
	return nil
}

// parseAndCompile parses the given patch contents,
// and compiles them into a gopatch program.
func parseAndCompile(fset *token.FileSet, name string, src []byte) (*engine.Program, error) {
	astProg, err := parse.Parse(fset, name, src)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	prog, err := engine.Compile(fset, astProg)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	return prog, nil
}
