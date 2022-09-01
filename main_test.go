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
					Patches: []string{"testdata/patch/error.patch"},
					Args:    arguments{Patterns: []string{"testdata/test_files/lint_example/test2.go"}},
					Diff:    true,
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
			})
		}
	})
}

func TestLoadPatches(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		opts := &options{Patches: []string{"testdata/patch/error.patch"},
			Diff:           true,
			DisplayVersion: false,
			Verbose:        false,
		}
		opts.Args.Patterns = []string{"testdata/test_files/lint_example/test2.go"}
		patch, err := loadPatches(token.NewFileSet(), opts, bytes.NewReader(nil))
		assert.Len(t, patch, 1)
		require.NoError(t, err)
	})
	t.Run("Success - mutiple patches", func(t *testing.T) {
		opts := &options{Patches: []string{"testdata/patch/error.patch"},
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
	t.Run("Failure", func(t *testing.T) {
		path := "testdata/patch/error1.patch"
		opts := &options{Patches: []string{path},
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
	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			name     string
			before   string
			after    string
			want     string
			filename string
		}{
			{
				name:     "preview",
				before:   "single line",
				after:    "different line",
				filename: "testdata/test_files/lint_example/time.go",
				want: "\n--- testdata/test_files/lint_example/time.go\n" +
					"+++ testdata/test_files/lint_example/time.go\n@@ -1,1 " +
					"+0,0 @@\n-single line\n@@ -0,0 +1,1 @@\n+different line\n",
			},
			{
				name:     "empty file",
				filename: "testdata/test_files/lint_example/time.go",
				want: "\n--- testdata/test_files/lint_example/time.go\n" +
					"+++ testdata/test_files/lint_example/time.go\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stderr, stdout bytes.Buffer
				comment := []string{"example comment"}
				err := preview(tt.filename, []byte(tt.before), []byte(tt.after), comment, &stderr, &stdout)
				require.NoError(t, err)
				outputString := stderr.String() + stdout.String()
				want := comment[0] + tt.want
				assert.Equal(t, want, outputString)
			})
		}

	})

	t.Run("nil for strings", func(t *testing.T) {
		comment := []string{"example comment"}
		filename := "testdata/test_files/lint_example/time.go"
		var stderr, stdout bytes.Buffer
		err := preview(filename, nil, nil, comment, &stderr, &stdout)
		require.NoError(t, err)
		outputString := stderr.String() + stdout.String()
		expectedString := comment[0] + "\n--- testdata/test_files/lint_example/time.go\n+++ " +
			"testdata/test_files/lint_example/time.go\n"
		assert.Equal(t, expectedString, outputString)
	})
}

func TestPrintComments(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tests := []struct {
			give []string
			name string
			want string
		}{
			{
				name: "single line comment",
				give: []string{"Since, gomock 1.5.0, Controller.Finish does not need to be called manually"},
				want: "Since, gomock 1.5.0, Controller.Finish does not need to be called manually\n",
			},
			{
				name: "multiple line comment",
				give: []string{"#This is the firstline of the comment", " This is",
					"a multiple line comment.", "This is the 3rd line of the comment"},
				want: "#This is the firstline of the comment\n" + " This is\n" +
					"a multiple line comment.\n" + "This is the 3rd line of the comment\n",
			},
			{
				name: "empty comment",
				give: []string{""},
				want: "\n",
			},
			{
				name: "nil comment",
				give: []string(nil),
				want: "",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stderr bytes.Buffer
				printComments(tt.give, &stderr)
				got := stderr.String()
				assert.Equal(t, tt.want, got)
			})
		}

	})
}
