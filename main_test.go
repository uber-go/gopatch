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
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArgParser(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			desc string
			give []string
			want options
		}{
			{
				desc: "-d",
				give: []string{"-d", "-p", "testdata/patch/error.patch", "testdata/test_files/lint_example/test2.go"},
				want: options{
					Patches: []string{"testdata/patch/error.patch"},
					Args:    arguments{Patterns: []string{"testdata/test_files/lint_example/test2.go"}},
					Diff:    true,
				},
			},
			{
				desc: "diff",
				give: []string{"--diff", "-p", "testdata/patch/error.patch", "testdata/test_files/lint_example/test2.go"},
				want: options{
					Patches:              []string{"testdata/patch/error.patch"},
					Args:                 arguments{Patterns: []string{"testdata/test_files/lint_example/test2.go"}},
					Diff:                 true,
					SkipImportProcessing: false,
				},
			},
			{
				desc: "skip-import-processing",
				give: []string{"--skip-import-processing", "-p", "testdata/patch/replace_to_with_ptr.patch", "testdata/test_files/skip_import_processing_example/test1.go"},
				want: options{
					Patches:              []string{"testdata/patch/replace_to_with_ptr.patch"},
					Args:                 arguments{Patterns: []string{"testdata/test_files/skip_import_processing_example/test1.go"}},
					SkipImportProcessing: true,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				argParser, opts := newArgParser()
				_, err := argParser.ParseArgs(tt.give)
				require.NoError(t, err)
				assert.Equal(t, tt.want.Patches, opts.Patches)
				assert.Equal(t, tt.want.Args, opts.Args)
				assert.Equal(t, tt.want.Diff, opts.Diff)
				assert.Equal(t, tt.want.SkipImportProcessing, opts.SkipImportProcessing)
			})
		}
	})
}

func TestLoadPatches(t *testing.T) {
	t.Parallel()

	t.Run("single patch", func(t *testing.T) {
		opts := &options{
			Patches:        []string{"testdata/patch/error.patch"},
			Diff:           true,
			DisplayVersion: false,
			Verbose:        false,
		}
		opts.Args.Patterns = []string{"testdata/test_files/lint_example/test2.go"}
		patch, err := loadPatches(token.NewFileSet(), opts, bytes.NewReader(nil))
		assert.Len(t, patch, 1)
		require.NoError(t, err)
	})

	t.Run("multiple patches", func(t *testing.T) {
		opts := &options{
			Patches:        []string{"testdata/patch/error.patch"},
			Diff:           true,
			DisplayVersion: false,
			Verbose:        false,
		}
		opts.Patches = []string{"testdata/patch/error.patch", "testdata/patch/time.patch"}
		opts.Args.Patterns = []string{"testdata/test_files/lint_example/"}
		patch, err := loadPatches(token.NewFileSet(), opts, bytes.NewReader(nil))
		require.NoError(t, err)
		assert.Equal(t, 2, len(patch))
	})

	t.Run("patches file", func(t *testing.T) {
		patchesFile := writeFile(t, filepath.Join(t.TempDir(), "patches.txt"),
			"testdata/patch/error.patch",
			"testdata/patch/time.patch",
		)
		opts := &options{
			PatchesFile: patchesFile,
			Diff:        true,
			Args: arguments{
				Patterns: []string{"testdata/test_files/lint_example/"},
			},
		}

		patch, err := loadPatches(token.NewFileSet(), opts, bytes.NewReader(nil))
		require.NoError(t, err)
		assert.Equal(t, 2, len(patch))
	})

	t.Run("error", func(t *testing.T) {
		path := "testdata/patch/error1.patch"
		opts := &options{
			Patches:        []string{path},
			Diff:           true,
			DisplayVersion: false,
			Verbose:        false,
		}
		opts.Args.Patterns = []string{"testdata/test_files/test2.go"}
		_, err := loadPatches(token.NewFileSet(), opts, bytes.NewReader(nil))
		assert.ErrorContains(t, err, path)
		assert.ErrorContains(t, err, "no such file or directory")
	})
}

func TestPreview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		before   string
		after    string
		want     string
		filename string
	}{
		{
			name:   "preview",
			before: "single line",
			after:  "different line",
			want: "--- testdata/test_files/lint_example/time.go\n" +
				"+++ testdata/test_files/lint_example/time.go\n" +
				"@@ -1,1 +0,0 @@\n" +
				"-single line\n" +
				"@@ -0,0 +1,1 @@\n" +
				"+different line\n",
		},
		{
			name: "empty file",
			want: "--- testdata/test_files/lint_example/time.go\n" +
				"+++ testdata/test_files/lint_example/time.go\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr, stdout bytes.Buffer
			cmd := &mainCmd{
				Stdout: &stdout,
				Stderr: &stderr,
				Getwd:  os.Getwd,
			}

			comment := []string{"example comment"}
			err := cmd.preview("testdata/test_files/lint_example/time.go", []byte(tt.before), []byte(tt.after), comment)
			require.NoError(t, err)

			assert.Equal(t, "testdata/test_files/lint_example/time.go:example comment\n", stderr.String())
			assert.Equal(t, tt.want, stdout.String())
		})
	}

	t.Run("nil for strings", func(t *testing.T) {
		var stderr, stdout bytes.Buffer
		cmd := &mainCmd{
			Stdout: &stdout,
			Stderr: &stderr,
			Getwd:  os.Getwd,
		}

		comment := []string{"example comment"}
		err := cmd.preview("testdata/test_files/lint_example/time.go", nil, nil, comment)
		require.NoError(t, err)

		assert.Equal(t, "testdata/test_files/lint_example/time.go:example comment\n", stderr.String())
		assert.Equal(t,
			"--- testdata/test_files/lint_example/time.go\n"+
				"+++ testdata/test_files/lint_example/time.go\n",
			stdout.String())
	})
}
