package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/uber-go/gopatch/internal/engine"
	"github.com/uber-go/gopatch/internal/parse"
	"github.com/jessevdk/go-flags"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
)

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdin); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	Patches []string `short:"p" long:"patch" value-name:"file"`
	Args    struct {
		Patterns []string `positional-arg-name:"pattern" required:"1"`
	} `positional-args:"yes"`
}

func newArgParser() (*flags.Parser, *options) {
	var opts options
	parser := flags.NewParser(&opts, flags.HelpFlag)
	parser.Name = "gopatch"

	// The following is more readable than long descriptions in struct
	// tags.

	parser.FindOptionByLongName("patch").Description =
		"Path to a patch file specifying the code transformation. " +
			"Multiple patches may be provided to be applied in-order. " +
			"If the flag is omitted, a patch will be read from stdin."

	parser.Args()[0].Description =
		"One or more Go package patterns to run the transformation against. " +
			"Patterns are absolute or relative import paths to specific packages " +
			"or import paths ending with ./... to transform a package and all its " +
			"descendants. Patterns can be paths to Go files to transform only those " +
			"files."

	return parser, &opts
}

func parseAndCompile(fset *token.FileSet, name string, src []byte) (*engine.Program, error) {
	astProg, err := parse.Parse(fset, name, src)
	if err != nil {
		return nil, err
	}
	return engine.Compile(fset, astProg)
}

func loadPatches(fset *token.FileSet, opts *options, stdin io.Reader) ([]*engine.Program, error) {
	if len(opts.Patches) == 0 {
		// No patches. Read from stdin.
		src, err := ioutil.ReadAll(stdin)
		if err != nil {
			return nil, err
		}

		prog, err := parseAndCompile(fset, "stdin", src)
		if err != nil {
			return nil, err
		}

		return []*engine.Program{prog}, err
	}

	// Load each patch provided with "-p".
	var progs []*engine.Program
	for _, path := range opts.Patches {
		src, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("could not read %q: %v", path, err)
		}

		prog, err := parseAndCompile(fset, path, src)
		if err != nil {
			return nil, fmt.Errorf("could not load patch %q: %v", path, err)
		}

		progs = append(progs, prog)
	}

	return progs, nil
}

func run(args []string, stdin io.Reader) error {
	argParser, opts := newArgParser()
	if _, err := argParser.ParseArgs(args); err != nil {
		return err
	}

	fset := token.NewFileSet()
	progs, err := loadPatches(fset, opts, stdin)
	if err != nil {
		return err
	}

	patchRunner := newPatchRunner(fset, progs)

	// We use packages.Load only to find the .go files. We parse them
	// separately to prevent having the ASTs of all files in memory at the
	// same time.
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadFiles,
	}, opts.Args.Patterns...)
	// TODO(abg): This should probably consider test files too.
	if err != nil {
		return err
	}

	var files []string
	for _, pkg := range pkgs {
		files = append(files, pkg.GoFiles...)
	}

	var errors []error
	for _, filename := range files {
		f, err := parser.ParseFile(fset, filename, nil /* src */, parser.AllErrors|parser.ParseComments)
		if err != nil {
			errors = append(errors, fmt.Errorf("could not parse %q: %v", filename, err))
			continue
		}

		f, ok := patchRunner.Apply(filename, f)
		// If at least one patch didn't match, there's nothing to do.
		if !ok {
			continue
		}

		var out bytes.Buffer
		if err := format.Node(&out, fset, f); err != nil {
			errors = append(errors, fmt.Errorf("failed to rewrite %q: %v", filename, err))
			continue
		}

		if err := ioutil.WriteFile(filename, out.Bytes(), 0644); err != nil {
			errors = append(errors, err)
			continue
		}
	}

	errors = append(errors, patchRunner.errors...)
	return multierr.Combine(errors...)
}

type patchRunner struct {
	fset    *token.FileSet
	patches []*engine.Program
	errors  []error
}

func newPatchRunner(fset *token.FileSet, patches []*engine.Program) *patchRunner {
	return &patchRunner{
		fset:    fset,
		patches: patches,
	}
}

func (r *patchRunner) Apply(filename string, f *ast.File) (*ast.File, bool) {
	matched := false
	for _, prog := range r.patches {
		for _, c := range prog.Changes {
			d, ok := c.Match(f)
			if !ok {
				// This patch didn't modify the file. Try the next one.
				continue
			}

			matched = true

			var err error
			f, err = c.Replace(d)
			if err != nil {
				r.errors = append(r.errors, fmt.Errorf("could not update %q: %v", filename, err))
				return nil, false
			}
		}
	}

	return f, matched
}
