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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/diff"
	"github.com/uber-go/gopatch/internal/astdiff"
	"github.com/uber-go/gopatch/internal/engine"
	"github.com/uber-go/gopatch/internal/parse"
	"go.uber.org/multierr"
	"golang.org/x/tools/imports"
)

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdin, os.Stderr, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

type arguments struct {
	Patterns []string `positional-arg-name:"pattern"`
}

type options struct {
	Patches        []string  `short:"p" long:"patch" value-name:"file"`
	Diff           bool      `short:"d" long:"diff"`
	DisplayVersion bool      `long:"version"`
	Print          bool      `long:"print-only"`
	Args           arguments `positional-args:"yes"`
	Verbose        bool      `short:"v" long:"verbose"`
}

func newArgParser() (*flags.Parser, *options) {
	var opts options
	parser := flags.NewParser(&opts, flags.HelpFlag)
	parser.Name = "gopatch"

	// The following is more readable than long descriptions in struct
	// tags.

	parser.FindOptionByLongName("version").Description =
		"Display the version of gopatch."

	parser.FindOptionByLongName("verbose").Description =
		"Turn on verbose mode that prints whether or not the file was patched " +
			"for each file found."

	parser.FindOptionByLongName("patch").Description =
		"Path to a patch file specifying the code transformation. " +
			"Multiple patches may be provided to be applied in-order. " +
			"If the flag is omitted, a patch will be read from stdin."

	parser.FindOptionByLongName("diff").Description =
		"Print a diff of the proposed changes to stdout but don't modify any files."

	parser.FindOptionByLongName("print-only").Description =
		"If true, don't modify the code and print the changed code to stdout."
	parser.Args()[0].Description =
		"One or more files or directores containing Go code. " +
			"When directories are provided, all Go files in them and their " +
			"descendants will be transformed."

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
		src, err := io.ReadAll(stdin)
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
		src, err := os.ReadFile(path)
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

func findGoFiles(path string) ([]string, error) {
	// Users may expect "./..."-stlye patterns to work.
	path = strings.TrimSuffix(path, "...")
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	var files []string
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		mode := info.Mode()
		switch {
		case mode.IsRegular():
			if strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}

		case mode.IsDir():
			base := filepath.Base(path)
			switch {
			case len(base) == 0,
				base[0] == '.',
				base[0] == '_',
				base == "testdata",
				base == "vendor":
				return filepath.SkipDir
			}
		}

		return nil
	})

	return files, err
}

func findFiles(patterns []string) (_ []string, err error) {
	files := make(map[string]struct{})
	for _, pat := range patterns {
		fs, findErr := findGoFiles(pat)
		if findErr != nil {
			err = multierr.Append(err, fmt.Errorf("enumerating Go files in %q: %v", pat, err))
			continue
		}

		for _, f := range fs {
			files[f] = struct{}{}
		}
	}

	sortedFiles := make([]string, 0, len(files))
	for f := range files {
		sortedFiles = append(sortedFiles, f)
	}
	sort.Strings(sortedFiles)

	return sortedFiles, err
}

func printComments(comments []string, stderr io.Writer) {
	for _, c := range comments {
		fmt.Fprintln(stderr, c)
	}
}

func preview(
	filename string,
	originalContent, modifiedContent []byte,
	comments []string,
	stderr, stdout io.Writer,
) error {
	printComments(comments, stderr)
	return diff.Text(filename, filename, originalContent, modifiedContent, stdout)
}

func run(args []string, stdin io.Reader, stderr io.Writer, stdout io.Writer) error {
	argParser, opts := newArgParser()
	if _, err := argParser.ParseArgs(args); err != nil {
		return err
	}
	if opts.DisplayVersion {
		fmt.Fprintln(stderr, "gopatch "+_version)
		return nil
	}

	if len(opts.Args.Patterns) == 0 {
		argParser.WriteHelp(stderr)
		fmt.Fprintln(stderr)

		return errors.New("please provide at least one pattern")
	}

	logOut := io.Discard
	if opts.Verbose {
		logOut = os.Stdout
	}
	log := log.New(logOut, "", 0)

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
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(fset, filename, content /* src */, parser.AllErrors|parser.ParseComments)
		if err != nil {
			log.Printf("%s: failed: %v", filename, err)
			errors = append(errors, fmt.Errorf("could not parse %q: %v", filename, err))
			continue
		}

		f, comments, ok := patchRunner.Apply(filename, f)
		// If at least one patch didn't match, there's nothing to do.
		if !ok {
			log.Printf("%s: skipped", filename)
			continue
		}

		var out bytes.Buffer
		if err := format.Node(&out, fset, f); err != nil {
			log.Printf("%s: failed: %v", filename, err)
			errors = append(errors, fmt.Errorf("failed to rewrite %q: %v", filename, err))
			continue
		}

		bs, err := imports.Process(filename, out.Bytes(), &imports.Options{
			Comments:   true,
			TabIndent:  true,
			TabWidth:   8,
			FormatOnly: true,
		})
		if err != nil {
			log.Printf("%s: failed: %v", filename, err)
			errors = append(errors, fmt.Errorf("reformat %q: %v", filename, err))
			continue
		}

		switch {
		case opts.Diff:
			err = preview(filename, content, bs, comments, stderr, stdout)
		case opts.Print:
			printComments(comments, stderr)
			_, err = stdout.Write(bs)
		default:
			err = os.WriteFile(filename, bs, 0644)
		}
		if err != nil {
			log.Printf("%s: failed: %v", filename, err)
			errors = append(errors, err)
			continue
		}
		log.Printf("%s: patched", filename)
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

func (r *patchRunner) Apply(filename string, f *ast.File) (fout *ast.File, comments []string, matched bool) {
	snap := astdiff.Before(f, ast.NewCommentMap(r.fset, f, f.Comments))

	for _, prog := range r.patches {
		for _, c := range prog.Changes {
			d, ok := c.Match(f)
			if !ok {
				// This patch didn't modify the file. Try the next one.
				continue
			}

			matched = true
			comments = c.Comments

			cl := engine.NewChangelog()

			var err error
			fout, err = c.Replace(d, cl)
			if err != nil {
				r.errors = append(r.errors, fmt.Errorf("could not update %q: %v", filename, err))
				return nil, comments, false
			}

			snap = snap.Diff(fout, cl)
			cleanupFilePos(r.fset.File(fout.Pos()), cl, fout.Comments)
		}
	}

	return fout, comments, matched
}

func cleanupFilePos(tfile *token.File, cl engine.Changelog, comments []*ast.CommentGroup) {
	linesToDelete := make(map[int]struct{})
	for _, dr := range cl.ChangedIntervals() {
		if dr.Start == token.NoPos {
			continue
		}

		for i := tfile.Line(dr.Start); i < tfile.Line(dr.End); i++ {
			if i > 0 {
				linesToDelete[i] = struct{}{}
			}
		}

		// Remove comments in the changed sections of the code.
		for _, cg := range comments {
			var list []*ast.Comment
			for _, c := range cg.List {
				if c.Pos() >= dr.Start && c.End() <= dr.End {
					continue
				}
				list = append(list, c)
			}
			cg.List = list
		}
	}

	lines := make([]int, 0, len(linesToDelete))
	for i := range linesToDelete {
		lines = append(lines, i)
	}
	sort.Ints(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		tfile.MergeLine(lines[i])
	}
}
