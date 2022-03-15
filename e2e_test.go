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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/txtar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestIntegration(t *testing.T) {
	const testdata = "testdata"

	infos, err := ioutil.ReadDir(testdata)
	require.NoErrorf(t, err, "failed to ls %q", testdata)

	// Disable Go modules so that we don't try to fetch against tests.
	oldModule := os.Getenv("GO111MODULE")
	os.Setenv("GO111MODULE", "off")
	defer os.Setenv("GO111MODULE", oldModule)

	for _, info := range infos {
		if info.IsDir() || info.Name() == "README.md" {
			continue
		}

		t.Run(info.Name(), func(t *testing.T) {
			runIntegrationTest(t, filepath.Join(testdata, info.Name()))
		})
	}
}

func resolvePatchPath(t *testing.T, path string) string {
	// If a patch file takes the form,
	//   => another/file.patch
	// Replace its path with the provided path.
	const linkPrefix = "=>"

	f, err := os.Open(path)
	require.NoError(t, err, "open patch %q", path)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return path
	}
	line := scanner.Text()

	// Not a link to anothe rfile. Return the original path
	// unchanged.
	if !strings.HasPrefix(line, linkPrefix) {
		return path
	}

	return strings.TrimSpace(strings.TrimPrefix(line, linkPrefix))
}

func runIntegrationTest(t *testing.T, testFile string) {
	ta, err := loadTestArchive(testFile)
	require.NoError(t, err, "failed to load tests from txtar")

	dir, err := ioutil.TempDir("", "gopatch-integration")
	require.NoError(t, err, "failed to create temporary directory")
	defer os.RemoveAll(dir)

	require.NoErrorf(t, txtar.Write(ta.Archive, dir),
		"could not write archive to %q", dir)

	var (
		args  []string // args for run()
		stdin []byte
	)
	// If there's only one patch, use stdin. Otherwise, use "-p".
	if ps := ta.Patches; len(ps) == 1 {
		path := resolvePatchPath(t, filepath.Join(dir, ps[0]))
		stdin, err = ioutil.ReadFile(path)
		require.NoError(t, err, "failed to read patch file %q", ps)
	} else {
		for _, path := range ps {
			path = resolvePatchPath(t, filepath.Join(dir, path))
			args = append(args, "-p", path)
		}
	}

	for _, tt := range ta.Files {
		t.Run(tt.Name, func(t *testing.T) {
			if skipTest(testFile, tt.Name) {
				t.Skipf("skipping unfixed test case %v/%v", testFile, tt.Name)
			}

			filePath := filepath.Join(dir, tt.Give)
			args := append([]string{filePath}, args...)
			require.NoError(t, run(args, bytes.NewReader(stdin), new(bytes.Buffer)),
				"could not run patch")

			got, err := ioutil.ReadFile(filePath)
			require.NoErrorf(t, err, "failed to read %q", filePath)
			assert.Equal(t, string(tt.Want), string(got))
		})
	}
}

func skipTest(testFile, testName string) bool {
	fullName := filepath.Join(filepath.Base(testFile), testName)
	_, skip := testsToSkip[fullName]
	return skip
}

// testsToSkip is a set of integration tests that do not yet pass, but
// eventually should. It is essentially a todo list of bugs to be fixed.
// For now, skip these tests.
var testsToSkip = map[string]struct{}{
	"add_error_param/a":              {}, // https://github.com/uber-go/gopatch/issues/6
	"func_within_a_func/two_dots":    {},
	"mismatched_dots/unnamed":        {}, // https://github.com/uber-go/gopatch/issues/9
	"switch_elision/body":            {},
	"select_elision/example":         {},
	"case_elision/a":                 {},
	"httpclient_use_ctx/one_return":  {},
	"const_to_var/single_top_level":  {},
	"struct_field_list/zero":         {},
	"struct_field_list/middle":       {},
	"value_group_elision/func":       {}, // https://github.com/uber-go/gopatch/issues/6
	"value_list_elision/foo":         {}, // https://github.com/uber-go/gopatch/issues/6
	"range_value_elision/no_params":  {},
	"range_value_elision/one_param":  {},
	"range_value_elision/two_params": {},
}

const (
	_patch = ".patch"
	_in    = ".in.go"
	_out   = ".out.go"
	_go    = ".go"
)

type testArchive struct {
	Archive *txtar.Archive

	// Names of files in the archive that are patches.
	Patches []string

	// Integration test files found in the patch.
	Files []*testFile
}

type testFile struct {
	// Name of the test file.
	//
	// Specified in the name of the go file before the .in.go/.out.go.
	Name string

	// Path to the input Go file.
	Give string

	// Expected contents of the Go file after the patches have been
	// applied.
	Want []byte
}

// Loads a test archive in-memory.
func loadTestArchive(path string) (*testArchive, error) {
	// test case name -> test case
	index := make(map[string]*testFile)

	// Retrieves test cases by name, adding to the map if needed.
	getTestFile := func(name string) *testFile {
		tf := index[name]
		if tf == nil {
			tf = &testFile{Name: name}
			index[name] = tf
		}
		return tf
	}

	archive, err := txtar.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %v", path, err)
	}

	// Rather than reproducing the entire contents of the txtar, we only
	// want the patches and input files. The contents of the output files
	// are needed in-memory.

	var patches []string // names of patch files

	newFiles := archive.Files[:0] // zero-alloc filtering
	for _, f := range archive.Files {
		switch {
		case strings.HasSuffix(f.Name, _patch):
			patches = append(patches, f.Name)

		case strings.HasSuffix(f.Name, _in):
			name := strings.TrimSuffix(f.Name, _in) // foo.in.go => foo

			// Replace the .in.go suffix with just .go so that
			// test cases have more control over the name of the
			// file when it affects the behavior of `go list`
			// (test files, for example).
			f.Name = name + _go // foo.in.go => foo.go
			f.Data = singleTrailingNewline(f.Data)

			getTestFile(name).Give = f.Name
		case strings.HasSuffix(f.Name, _out):
			name := strings.TrimSuffix(f.Name, _out) // foo.out.go => foo
			getTestFile(name).Want = singleTrailingNewline(f.Data)

			// Don't include this file in the list of files
			// reproduced by the archive.
			continue
		default:
			err = multierr.Append(err,
				fmt.Errorf("unknown file %q found in %q", f.Name, path))
		}

		newFiles = append(newFiles, f)
	}
	archive.Files = newFiles

	if len(patches) == 0 {
		err = multierr.Append(err, fmt.Errorf("no patches found in %q", path))
	}

	if len(index) == 0 {
		err = multierr.Append(err, fmt.Errorf("no Go files found in %q", path))
	}

	files := make([]*testFile, 0, len(index))
	for _, tt := range index {
		if len(tt.Give) == 0 {
			err = multierr.Append(err, fmt.Errorf(
				"test %q of %q does not have an input file", tt.Name, path))
		}
		if len(tt.Want) == 0 {
			err = multierr.Append(err, fmt.Errorf(
				"test %q of %q does not have an output file", tt.Name, path))
		}
		files = append(files, tt)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	return &testArchive{
		Archive: archive,
		Patches: patches,
		Files:   files,
	}, nil
}

// Removes all but the last trailing newline from a slice.
//
// Makes for easier to read test cases.
func singleTrailingNewline(bs []byte) []byte {
	i := len(bs) - 1
	for i > 0 && bs[i] == '\n' && bs[i-1] == '\n' {
		i--
	}
	return bs[:i+1]
}
