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
	"path/filepath"
	"sort"
	"strings"

	"github.com/uber-go/gopatch/internal/engine"
	"github.com/uber-go/gopatch/internal/parse"
	"github.com/jessevdk/go-flags"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/packages"
)

func main() {
	log.SetFlags(0)
	// TODO: proper CLI parsing
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

func findFiles(patterns []string) ([]string, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedFiles,
	}, patterns...)
	if err != nil {
		return nil, err
	}

	files := make(map[string]struct{})
	for _, pkg := range pkgs {
		dirs := make(map[string]struct{})
		for _, f := range pkg.GoFiles {
			dirs[filepath.Dir(f)] = struct{}{}
			files[f] = struct{}{}
		}

		// We enumerate the test files manually. We need to do this
		// because we can't use the Tests flag of packages.Config due
		// to a limitation of `go list`: It doesn't accept `-test` and
		// `-find` at the same time[1].
		//
		// [1] https://github.com/golang/tools/blob/master/go/packages/golist.go#L837
		//
		// Without the `-find` flag, `go list` requires that all code
		// compile, or at least all imports be present on the GOPATH.
		// This is a problem for gopatch because one of our
		// requirements is being able to operate on code that doesn't
		// compile.
		//
		// So we'll enumerate the test files in each directory
		// explicitly.
		for dir := range dirs {
			infos, err := ioutil.ReadDir(dir)
			if err != nil {
				return nil, fmt.Errorf("could not ls %q: %v", dir, err)
			}
			for _, i := range infos {
				if i.IsDir() {
					continue
				}
				name := i.Name()
				if !strings.HasSuffix(name, "_test.go") {
					continue
				}
				files[filepath.Join(dir, name)] = struct{}{}
			}
		}
	}

	sortedFiles := make([]string, 0, len(files))
	for f := range files {
		sortedFiles = append(sortedFiles, f)
	}
	sort.Strings(sortedFiles)
	return sortedFiles, nil
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

	files, err := findFiles(opts.Args.Patterns)
	if err != nil {
		return err
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
