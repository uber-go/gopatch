package main

import (
	"bytes"
	"container/list"
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

	for _, info := range infos {
		if info.IsDir() || info.Name() == "README.md" {
			continue
		}

		t.Run(info.Name(), func(t *testing.T) {
			runIntegrationTest(t, filepath.Join(testdata, info.Name()))
		})
	}
}

func runIntegrationTest(t *testing.T, testFile string) {
	tc, err := loadIntegrationTestCases(testFile)
	require.NoError(t, err, "failed to load tests from txtar")
	defer tc.Cleanup()

	var (
		args  []string // args for run()
		stdin []byte
	)
	// If there's only one patch, use stdin. Otherwise, use "-p".
	if len(tc.Patches) == 1 {
		stdin, err = ioutil.ReadFile(tc.Patches[0])
		require.NoError(t, err, "failed to read patch file %q", tc.Patches[0])
	} else {
		for _, patch := range tc.Patches {
			args = append(args, "-p", patch)
		}
	}

	for _, tt := range tc.Files {
		t.Run(tt.Name, func(t *testing.T) {
			args := append([]string{tt.Give}, args...)
			require.NoError(t, run(args, bytes.NewReader(stdin)),
				"could not run patch")

			got, err := ioutil.ReadFile(tt.Give)
			require.NoErrorf(t, err, "failed to read %q", tt.Give)
			assert.Equal(t, string(tt.Want), string(got))
		})
	}
}

const (
	_patch = ".patch"
	_in    = ".in.go"
	_out   = ".out.go"
)

type integrationTestCase struct {
	// Paths to patch files containing the patches to be applied.
	Patches []string

	// Go files that will be verified.
	Files []*integrationTestFile

	cleanup func()
}

// Cleanup cleans up resources acquired by the test caes.
func (tc *integrationTestCase) Cleanup() {
	if tc.cleanup != nil {
		tc.cleanup()
	}
}

type integrationTestFile struct {
	// Name of the test case. Specified in the name of the go file before
	// the .in.go/.out.go.
	Name string

	// Path to the .in.go file.
	Give string

	// Expected contents of the .out.go file.
	Want []byte
}

// Loads the integration test cases from a single txtar file.
//
// The cleanup function on the returned test case MUST be called regardless of
// whether the test succeeeded.
func loadIntegrationTestCases(testFile string) (tc integrationTestCase, err error) {
	var es exitStack
	defer func() {
		es.ExitOnPanic()

		// No panic if we get here. Clean up immediately if we saw an
		// error. Otherwise, inform the test case how to cleanup.
		if err != nil {
			es.Exit()
		} else {
			tc.cleanup = es.Exit
		}
	}()

	dir, err := ioutil.TempDir("", "gopatch-integration")
	if err != nil {
		return tc, fmt.Errorf("failed to create temporary directory")
	}
	es.Defer(func() { os.RemoveAll(dir) })

	archive, err := txtar.ParseFile(testFile)
	if err != nil {
		return tc, fmt.Errorf("failed to open archive %q: %v", testFile, err)
	}

	if err := txtar.Write(archive, dir); err != nil {
		return tc, fmt.Errorf("failed to write archive %q to directory %q: %v", testFile, dir, err)
	}

	// test case name -> test case
	index := make(map[string]*integrationTestFile)

	// Retrieves test cases by name, adding to the map if needed.
	getTestCase := func(name string) *integrationTestFile {
		tc := index[name]
		if tc == nil {
			tc = &integrationTestFile{Name: name}
			index[name] = tc
		}
		return tc
	}

	for _, f := range archive.Files {
		switch {
		case strings.HasSuffix(f.Name, _patch):
			tc.Patches = append(tc.Patches, filepath.Join(dir, f.Name))
		case strings.HasSuffix(f.Name, _in):
			name := strings.TrimSuffix(f.Name, _in)
			getTestCase(name).Give = filepath.Join(dir, f.Name)
		case strings.HasSuffix(f.Name, _out):
			name := strings.TrimSuffix(f.Name, _out)
			getTestCase(name).Want = f.Data
		default:
			err = multierr.Append(err, fmt.Errorf("unknown file %q found in %q", f.Name, testFile))
		}
	}

	if len(tc.Patches) == 0 {
		err = multierr.Append(err, fmt.Errorf("no patches found in %q", testFile))
	}
	if len(index) == 0 {
		err = multierr.Append(err, fmt.Errorf("no test cases found in %q", testFile))
	}

	tc.Files = make([]*integrationTestFile, 0, len(index))
	for _, tt := range index {
		if len(tt.Give) == 0 {
			err = multierr.Append(err, fmt.Errorf(
				"test %q of %q does not have an input file", tt.Name, testFile))
		}
		if len(tt.Want) == 0 {
			err = multierr.Append(err, fmt.Errorf(
				"test %q of %q does not have an output file", tt.Name, testFile))
		}
		tc.Files = append(tc.Files, tt)
	}
	sort.Slice(tc.Files, func(i, j int) bool {
		return tc.Files[i].Name < tc.Files[j].Name
	})

	return tc, err
}

// exitStack is a user-level implementation of Go's defer stack.
//
//   var es exitStack
//   defer es.Exit()
//
//   es.Defer(file.Close)
//   if foo {
//     es.Defer(func() { os.Remove(file) })
//   }
//
// The advantage of this structure is that we're able to decide whether
// deferred operations should be executed or not. For example, the following
// function runs cleaup in case of errors, but returns the cleanup function
// otherwise for the caller to do it as needed.
//
//   func foo() (cleanup func(), err error) {
//     var es exitStack
//     func() {
//       if err != nil { es.Exit() }
//       else { cleanup = es.Exit }
//     }()
//
//     ...
//   }
//
// The zero-value of exitStack is valid.
type exitStack struct{ funcs *list.List }

// Defer requests that this function be executed when this stack exits.
func (es *exitStack) Defer(f func()) {
	if es.funcs == nil {
		es.funcs = list.New()
	}

	es.funcs.PushFront(f)
}

// Exit runs all deferred functions in reverse order.
func (es *exitStack) Exit() {
	fs := es.funcs
	if fs == nil {
		return
	}

	for fs.Len() > 0 {
		f := fs.Remove(fs.Front()).(func())
		f()
	}
}

// ExitOnPanic runs the deferred functions only if the caller of this function
// panicked, and re-panics.
//
//   var es exitStack
//   defer es.ExitOnPanic()
//   ...
func (es *exitStack) ExitOnPanic() {
	if err := recover(); err != nil {
		es.Exit()
		panic(err)
	}
}
