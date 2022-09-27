package main

import (
	"errors"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/gopatch/internal/engine"
)

// fakeParseAndCompile builds a mock implementation of a function
// matching the signature of parseAndCompile.
//
//	newFakeParseAndCompile(t).
//		Expect(...).
//		Return(...).
//		Build()
type fakeParseAndCompile struct {
	t     *testing.T
	calls []*loadCall // list of calls expected in-order
}

// newFakeParseAndCompile builds a new fakeParseAndCompile mock.
func newFakeParseAndCompile(t *testing.T) *fakeParseAndCompile {
	return &fakeParseAndCompile{t: t}
}

// Expect specifies that the mock should expect a calls
// with the given file name and body.
//
// Expect returns a reference to the call object to allow specifying
// the behavior for that expected call.
func (m *fakeParseAndCompile) Expect(name string, body string) *loadCall {
	call := &loadCall{parent: m, name: name, body: body}
	m.calls = append(m.calls, call)
	return call
}

// Build builds the mock expectations into a function
// that can be set on patchLoader.
func (m *fakeParseAndCompile) Build() func(fs *token.FileSet, name string, body []byte) (*engine.Program, error) {
	t := m.t
	return func(fs *token.FileSet, name string, body []byte) (*engine.Program, error) {
		require.NotEmpty(t, m.calls, "unexpected call parseAndCompile(%v, %q, %q)", name, body)
		call := m.calls[0]
		m.calls = m.calls[1:]

		assert.Equal(t, call.name, name, "unexpected file name")
		assert.Equal(t, call.body, string(body), "unexpected file body")
		return new(engine.Program), call.err
	}
}

// loadCall is a single call expected by the mock.
type loadCall struct {
	parent *fakeParseAndCompile
	name   string
	body   string
	err    error
}

// Return specifies that the mock call should return the given error.
//
// Return yields a reference to the mock itself to aid in chaining calls.
func (c *loadCall) Return(err error) *fakeParseAndCompile {
	c.err = err
	return c.parent
}

func TestPatchLoader_LoadReader(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect("stdin", "hello").
			Return(nil).
			Build()

		err := loader.LoadReader("stdin", strings.NewReader("hello"))
		assert.NoError(t, err)
		assert.NotEmpty(t, loader.Programs())
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()

		giveErr := errors.New("great sadness")

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).Build()

		err := loader.LoadReader("stdin", iotest.ErrReader(giveErr))
		assert.ErrorContains(t, err, "read:")
		assert.ErrorIs(t, err, giveErr)
	})

	t.Run("load error", func(t *testing.T) {
		t.Parallel()

		giveErr := errors.New("great sadness")

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect("stdin", "world").
			Return(giveErr).
			Build()

		err := loader.LoadReader("stdin", strings.NewReader("world"))
		assert.ErrorIs(t, err, giveErr)
	})
}

func TestPatchLoader_LoadFile(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		file := writeFile(t, filepath.Join(t.TempDir(), "foo.patch"), "fix everything")

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect(file, "fix everything\n").
			Return(nil).
			Build()

		require.NoError(t, loader.LoadFile(file))
		assert.NotEmpty(t, loader.Programs())
	})

	t.Run("open error", func(t *testing.T) {
		t.Parallel()

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).Build()

		err := loader.LoadFile("some/file/that/does/not/exist.patch")
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("load error", func(t *testing.T) {
		t.Parallel()

		file := writeFile(t, filepath.Join(t.TempDir(), "foo.patch"), "fix everything")

		giveErr := errors.New("great sadness")

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect(file, "fix everything\n").
			Return(giveErr).
			Build()

		err := loader.LoadFile(file)
		assert.ErrorIs(t, err, giveErr)
	})
}

func TestPatchLoader_LoadFileList(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		fooPatch := writeFile(t, filepath.Join(dir, "foo.patch"), "fix foo")
		barPatch := writeFile(t, filepath.Join(dir, "bar.patch"), "fix bar")
		bazPatch := writeFile(t, filepath.Join(dir, "baz.patch"), "fix baz")
		patchList := writeFile(t, filepath.Join(dir, "files.txt"),
			fooPatch,
			"",
			barPatch,
			bazPatch,
			"", // extraneous empty lines
		)

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect(fooPatch, "fix foo\n").Return(nil).
			Expect(barPatch, "fix bar\n").Return(nil).
			Expect(bazPatch, "fix baz\n").Return(nil).
			Build()

		require.NoError(t, loader.LoadFileList(patchList))
	})

	t.Run("open error", func(t *testing.T) {
		t.Parallel()

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).Build()

		err := loader.LoadFileList("some/file/that/does/not/exist.patch")
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("load error", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		goodFile := writeFile(t, filepath.Join(dir, "good.patch"), "hi")
		badFile := writeFile(t, filepath.Join(dir, "bad.patch"), "bye")
		patchList := writeFile(t, filepath.Join(dir, "files.txt"),
			goodFile,
			badFile,
			filepath.Join(dir, "does_not_exist.patch"), // never called
		)

		giveErr := errors.New("great sadness")

		loader := newPatchLoader(token.NewFileSet())
		loader.parseAndCompile = newFakeParseAndCompile(t).
			Expect(goodFile, "hi\n").Return(nil).
			Expect(badFile, "bye\n").Return(giveErr).
			Build()

		err := loader.LoadFileList(patchList)
		assert.ErrorIs(t, err, giveErr)
		assert.ErrorContains(t, err, badFile)
	})
}

// writeFile writes a file with the given lines and returns its path.
func writeFile(t *testing.T, path string, lines ...string) string {
	t.Helper()

	contents := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644), "writeFile(%q)", path)
	return path
}
