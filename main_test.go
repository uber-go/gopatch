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

// Package section is responsible for splitting a program into its different
// sections without attempting to parse the contents.s
package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Arguments struct {
	args []string
	fset *token.FileSet
}

var command = Arguments{
	args: []string{"-l", "-p", "testdata/patch/error.patch", "testdata/test_files/lint_example/test2.go"},
	fset: token.NewFileSet(),
}

func TestNewArgParser(t *testing.T) {
	t.Parallel()
	t.Run("Verify correct initialization through newArgParser", func(t *testing.T) {
		argParser, opts := newArgParser()
		outputStringLint := argParser.FindOptionByLongName("lint").Description
		outputStringLintShort := argParser.FindOptionByShortName(rune('l')).Description
		expectedStringLint := "Turn on lint flag to output patch matches to stdout instead of modifying the files"
		assert.Equal(t, expectedStringLint, outputStringLint)
		assert.Equal(t, expectedStringLint, outputStringLintShort)
		assert.Equal(t, false, opts.Linting)
		assert.Equal(t, []string(nil), opts.Patches)

	})

	t.Run("Verify functionality of newArgParser", func(t *testing.T) {
		argParser, opts := newArgParser()
		_, err := argParser.ParseArgs(command.args)
		patchFileName := command.args[2]
		assert.Equal(t, true, opts.Linting)
		assert.Equal(t, []string{patchFileName}, opts.Patches)
		require.NoError(t, err)

	})

}

func TestLoadPatches(t *testing.T) {
	t.Parallel()
	t.Run("Verify correct functionality of LoadPatches", func(t *testing.T) {
		argParser, opts := newArgParser()
		_, err := argParser.ParseArgs(command.args)
		require.NoError(t, err)
		_, err = loadPatches(command.fset, opts, os.Stdin)
		require.NoError(t, err)
	})
	t.Run("Verify correct error gets thrown when loadPatches Failure", func(t *testing.T) {
		argParser, opts := newArgParser()
		_, err := argParser.ParseArgs([]string{"-l", "-p",
			"testdata/patch/error1.patch", "testdata/test_files/test2.go"})
		require.NoError(t, err)
		path := "testdata/patch/error1.patch"
		_, err = loadPatches(command.fset, opts, os.Stdin)
		expectedErrorString := "could not read \"" + path + "\": " + "open " + path + ": no such file or directory"
		require.EqualError(t, err, expectedErrorString)
	})
}

func TestApply(t *testing.T) {
	t.Parallel()
	t.Run("Verify functionality of Apply when successful", func(t *testing.T) {
		argParser, opts := newArgParser()
		_, err := argParser.ParseArgs(command.args)
		filename := "testdata/test_files/lint_example/test2.go"
		content, err := ioutil.ReadFile(filename)
		fast, err := parser.ParseFile(command.fset, filename, content /* src */, parser.AllErrors|parser.ParseComments)
		progs, err := loadPatches(command.fset, opts, os.Stdin)
		patchRunner := newPatchRunner(command.fset, progs)
		_, actualComments, _ := patchRunner.Apply(filename, fast)
		expectedComments := []string{" This patch replaces instances of fmt.Sprintf()",
			" with fmt.Errorf()", " Patch files can be applied to mutiple files"}
		require.NoError(t, err)
		assert.Equal(t, expectedComments, actualComments)
	})
	t.Run("Verify Apply functionality when patch doesn't apply", func(t *testing.T) {
		_, opts := newArgParser()
		filename := "testdata/test_files/lint_example/time.go"
		content, _ := ioutil.ReadFile(filename)
		fast, _ := parser.ParseFile(command.fset, filename, content /* src */, parser.AllErrors|parser.ParseComments)
		progs, _ := loadPatches(command.fset, opts, os.Stdin)
		patchRunner := newPatchRunner(command.fset, progs)
		_, actualComments, ok := patchRunner.Apply(filename, fast)
		expectedComments := []string(nil)
		assert.Equal(t, expectedComments, actualComments)
		assert.Equal(t, false, ok)
	})

}

func TestPreviewer(t *testing.T) {
	t.Parallel()
	t.Run("run functionality with -l", func(t *testing.T) {
		var tests = []struct {
			name           string
			give           []string
			want           string
			targetFilename string
		}{
			{
				name:           "Verify preview's functionality",
				give:           []string{"single line", "different line"},
				targetFilename: "testdata/test_files/lint_example/time.go",
				want: "\n" + "--- testdata/test_files/lint_example/time.go\n" +
					"+++ testdata/test_files/lint_example/time.go\n@@ -1,1 " +
					"+0,0 @@\n-single line\n@@ -0,0 +1,1 @@\n+different line\n",
			},
			{
				name:           "Verify preview's functionality - empty file",
				give:           []string{"", ""},
				targetFilename: "testdata/test_files/lint_example/time.go",
				want: "\n" + "--- testdata/test_files/lint_example/time.go\n" +
					"+++ testdata/test_files/lint_example/time.go\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := bytes.Buffer{}
				comment := []string{"example comment"}
				err := preview(tt.targetFilename, []byte(tt.give[0]), []byte(tt.give[1]), comment, &buf)
				outputString := buf.String()
				tt.want = strings.Join(comment, "\n") + tt.want
				require.NoError(t, err)
				assert.Equal(t, tt.want, outputString)
			})
		}

	})

	t.Run("Verify preview's functionality - nil for strings", func(t *testing.T) {
		comment := []string{"example comment"}
		filename := "testdata/test_files/lint_example/time.go"

		buf := bytes.Buffer{}
		err := preview(filename, nil, nil, comment, &buf)
		outputString := buf.String()
		expectedString := strings.Join(comment, "/n") + "\n" + "--- testdata/test_files/lint_example/time.go\n+++ " +
			"testdata/test_files/lint_example/time.go\n"
		require.NoError(t, err)
		assert.Equal(t, expectedString, outputString)
	})
}

func TestModifier(t *testing.T) {
	t.Parallel()
	t.Run("Verify modify's functionality", func(t *testing.T) {
		filename := "testdata/test_files/patch_examples/patcherTester.go"
		newLine := []byte("different line")
		err := modify(filename, newLine)
		require.NoError(t, err)
		modifiedFileContent, _ := ioutil.ReadFile(filename)
		assert.Equal(t, newLine, modifiedFileContent)
		err = modify(filename, []byte("package patch_examples\n"))
		require.NoError(t, err)
	})
	t.Run("Verify modify's functionality - input empty string", func(t *testing.T) {
		filename := "testdata/test_files/patch_examples/patcherTester.go"
		newLine := []byte("")
		err := modify(filename, newLine)
		require.NoError(t, err)
		modifiedFileContent, _ := ioutil.ReadFile(filename)
		assert.Equal(t, newLine, modifiedFileContent)
		err = modify(filename, []byte("package patch_examples\n"))
		require.NoError(t, err)
	})
	t.Run("Verify modify's functionality - input nil", func(t *testing.T) {
		filename := "testdata/test_files/patch_examples/patcherTester.go"
		err := modify(filename, nil)
		require.NoError(t, err)
		modifiedFileContent, _ := ioutil.ReadFile(filename)
		assert.Equal(t, []byte(""), modifiedFileContent)
		err = modify(filename, []byte("package patch_examples\n"))
		require.NoError(t, err)
	})
}
func TestPreviewEndToEnd(t *testing.T) {
	t.Parallel()
	argParser, opts := newArgParser()
	t.Run("Check Linting option displayed correctly", func(t *testing.T) {
		outputStringLint := argParser.FindOptionByLongName("lint").Description
		outputStringLintShort := argParser.FindOptionByShortName(rune('l')).Description
		expectedStringLint := "Turn on lint flag to output patch matches to stdout instead of modifying the files"
		assert.Equal(t, expectedStringLint, outputStringLint)
		assert.Equal(t, expectedStringLint, outputStringLintShort)
		assert.Equal(t, false, opts.Linting)
		assert.Equal(t, []string(nil), opts.Patches)
		assert.Equal(t, false, opts.Linting)

	})
	t.Run("Verify functionality of findFiles", func(t *testing.T) {
		_, err := findFiles(opts.Args.Patterns)
		require.NoError(t, err)
	})

	t.Run("run functionality with -l", func(t *testing.T) {
		var tests = []struct {
			name               string   // name
			give               []string // give
			comment            []string
			targetFilename     []string
			comparisonFilename []string
		}{
			{
				name: "Linting single patch - single target file",
				give: []string{"-l", "-p", "testdata/patch/time.patch", "testdata/test_files/lint_example/time.go"},
				comment: []string{" This patch replaces instances of time.Now().Sub(x)\n with time.Since(x) " +
					"where x is an identifier variable\n"},
				targetFilename:     []string{"testdata/test_files/lint_example/time.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/timeModified.go"},
			},
			{
				name: "Linting single patch - single target file difffernt -l placement",
				give: []string{"-p", "testdata/patch/error.patch", "-l", "testdata/test_files/lint_example/test2.go"},
				comment: []string{" This patch replaces instances of fmt.Sprintf()\n " +
					"with fmt.Errorf()\n Patch files can be applied to mutiple files\n"},
				targetFilename:     []string{"testdata/test_files/lint_example/test2.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/test2Modified.go"},
			},
			{
				name: "Linting multiple patch - single target file",
				give: []string{"-l", "-p", "testdata/patch/multiplePatch.patch", "testdata/test_files/lint_example/test2.go"},
				comment: []string{" This patch replaces instances of fmt.Sprintf()\n with fmt.Errorf()\n " +
					"Patch files can be applied to mutiple files\n"},
				targetFilename:     []string{"testdata/test_files/lint_example/test2.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/test2Modified.go"},
			},
			{
				name: "Linting multiple patch - target directory",
				give: []string{"-l", "-p", "testdata/patch/multiplePatch.patch", "testdata/test_files/lint_example/"},
				comment: []string{" This patch replaces instances of fmt.Sprintf()\n with fmt.Errorf()\n " +
					"Patch files can be applied to mutiple files\n", " This patch replaces instances " +
					"of fmt.Sprintf()\n with fmt.Errorf()\n Patch files can be applied to mutiple files\n",
					" This patch replaces instances of time.Now().Sub(x)\n with time.Since(x) " +
						"where x is an identifier variable\n"},
				targetFilename: []string{"testdata/test_files/lint_example/error.go",
					"testdata/test_files/lint_example/test2.go", "testdata/test_files/lint_example/time.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/errorModified.go",
					"testdata/test_files/lint_example/test2Modified.go",
					"testdata/test_files/lint_example/timeModified.go"},
			},
			{
				name: "Linting single patch - target directory",
				give: []string{"-l", "-p", "testdata/patch/error.patch", "testdata/test_files/lint_example/"},
				comment: []string{" This patch replaces instances of fmt.Sprintf()\n with fmt.Errorf()\n Patch files " +
					"can be applied to mutiple files\n", " This patch replaces instances of fmt.Sprintf()\n with " +
					"fmt.Errorf()\n Patch files can be applied to mutiple files\n"},
				targetFilename: []string{"testdata/test_files/lint_example/error.go",
					"testdata/test_files/lint_example/test2.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/errorModified.go",
					"testdata/test_files/lint_example/test2Modified.go"},
			},
			{
				name: "Linting -l flag places anywhere",
				give: []string{"-p", "testdata/patch/error.patch", "-l", "testdata/test_files/lint_example/"},
				comment: []string{" This patch replaces instances of fmt.Sprintf()\n with fmt.Errorf()\n Patch files " +
					"can be applied to mutiple files\n", " This patch replaces instances of fmt.Sprintf()\n with " +
					"fmt.Errorf()\n Patch files can be applied to mutiple files\n"},
				targetFilename: []string{"testdata/test_files/lint_example/error.go",
					"testdata/test_files/lint_example/test2.go"},
				comparisonFilename: []string{"testdata/test_files/lint_example/errorModified.go",
					"testdata/test_files/lint_example/test2Modified.go"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bufCommand := bytes.Buffer{}
				err := run(tt.give, os.Stdin, &bufCommand)
				require.NoError(t, err)
				outputString := bufCommand.String()
				want := ""
				for i := range tt.comment {
					fs, _ := findGoFiles(tt.targetFilename[i])
					content, _ := ioutil.ReadFile(fs[0])
					modifiedContent, _ := ioutil.ReadFile(tt.comparisonFilename[i])
					bufDiff := bytes.Buffer{}
					err = diff.Text(fs[0], fs[0], content, modifiedContent, &bufDiff)
					require.NoError(t, err)
					want += tt.comment[i] + bufDiff.String()
				}
				require.NoError(t, err)
				assert.Equal(t, want, outputString)
			})
		}

	})
	t.Run("run linting failing - wrong -l placement", func(t *testing.T) {
		give := []string{"-p", "-l", "testdata/patch/error.patch", "testdata/test_files/lint_example/test2.go"}
		buf := bytes.Buffer{}
		err := run(give, os.Stdin, &buf)
		expectedErrorString := "expected argument for flag `-p, --patch', but got option `-l'"
		require.EqualError(t, err, expectedErrorString)

	})

}

func TestPatchModifySuccess_run(t *testing.T) {
	t.Run("run functionality with -p", func(t *testing.T) {
		var tests = []struct {
			name               string
			give               []string
			targetFileName     string
			comparisonFileName string
		}{
			{
				name:               "Applying Patching single patch on single target file",
				give:               []string{"-p", "testdata/patch/error.patch", "testdata/test_files/patch_examples/test1.go"},
				targetFileName:     "testdata/test_files/patch_examples/test1.go",
				comparisonFileName: "testdata/test_files/patch_examples/test1AfterPatchFile.go",
			},
			{
				name: "Applying Patching multiple patch on single target file",
				give: []string{"-p", "testdata/patch/multiplePatch.patch",
					"testdata/test_files/patch_examples/test2.go"},
				targetFileName:     "testdata/test_files/patch_examples/test2.go",
				comparisonFileName: "testdata/test_files/patch_examples/test2AfterPatch.go",
			},
			{
				name:               "Applying Patching multiple patches on target directory",
				give:               []string{"-p", "testdata/patch/multiplePatch.patch", "testdata/test_files/patch_examples/"},
				targetFileName:     "testdata/test_files/patch_examples/timeBefore.go",
				comparisonFileName: "testdata/test_files/patch_examples/timeAfterPatch.go",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := bytes.Buffer{}
				err := run(tt.give, os.Stdin, &buf)
				require.NoError(t, err)
				err = diff.Text(tt.targetFileName, tt.comparisonFileName, nil, nil, &buf)
				require.NoError(t, err)
				outputString := buf.String()
				want := "--- " + tt.targetFileName + "\n+++ " + tt.comparisonFileName + "\n"
				assert.Equal(t, want, outputString)
				err = run([]string{"-p", "testdata/patch/reverseChanges.patch",
					tt.targetFileName}, os.Stdin, &buf)
				require.NoError(t, err)
			})
		}
	})
}
