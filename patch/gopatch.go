package patch

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"golang.org/x/tools/imports"
	"sort"

	"github.com/uber-go/gopatch/internal/astdiff"
	"github.com/uber-go/gopatch/internal/engine"
	"github.com/uber-go/gopatch/internal/parse"
)

// PatchFile is a patch difference that can be applied to Go file.
type PatchFile struct {
	fset *token.FileSet
	prog *engine.Program
}

// Parse the patch file and creates data that can be applied to the Go file.
func Parse(patchFileName string, src []byte) (*PatchFile, error) {
	fset := token.NewFileSet()

	astProg, err := parse.Parse(fset, patchFileName, src)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	prog, err := engine.Compile(fset, astProg)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}

	return &PatchFile{fset: fset, prog: prog}, nil
}

// Apply takes the Go file name and its contents and returns a Go file with the patch applied.
func (patchFile *PatchFile) Apply(filename string, src []byte) ([]byte, error) {
	f, err := parser.ParseFile(patchFile.fset, filename, src, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("could not parse %q: %w", filename, err)
	}

	snap := astdiff.Before(f, ast.NewCommentMap(patchFile.fset, f, f.Comments))

	var fout *ast.File
	var retErr error
	for _, c := range patchFile.prog.Changes {
		d, ok := c.Match(f)
		if !ok {
			// This patch didn't modify the file. Try the next one.
			continue
		}

		cl := engine.NewChangelog()

		fout, err = c.Replace(d, cl)
		if err != nil {
			retErr = errors.Join(retErr, err)
			continue
		}

		snap = snap.Diff(fout, cl)
		cleanupFilePos(patchFile.fset.File(fout.Pos()), cl, fout.Comments)
	}

	if retErr != nil {
		return nil, retErr
	}

	if fout == nil {
		return src, nil
	}

	var out bytes.Buffer
	err = format.Node(&out, patchFile.fset, fout)
	if err != nil {
		return nil, err
	}

	bs := out.Bytes()
	bs, err = imports.Process(filename, bs, &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: true,
	})
	if err != nil {
		return nil, err
	}

	return bs, nil
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
