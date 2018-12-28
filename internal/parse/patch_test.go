package parse

import (
	"bytes"
	"fmt"
	"go/token"
	"testing"

	"github.com/uber-go/gopatch/internal/parse/section"
	"github.com/uber-go/gopatch/internal/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeparatePatch(t *testing.T) {
	// Build up a fake change.
	var buff bytes.Buffer
	fmt.Fprintln(&buff, "@@")
	fmt.Fprintln(&buff, "@@")

	buff.Write(text.Unlines(
		" foo",
		"-bar",
		"+baz",
		"qux",
	))

	contents := buff.Bytes()

	fset := token.NewFileSet()
	prog, err := section.Split(fset, "test.patch", contents)
	require.NoError(t, err, "failed to split sections")
	require.Len(t, prog, 1, "expected exactly one change")

	before, after := splitPatch(prog[0].Patch)

	assert.Equal(t,
		text.Unlines(" foo", "bar", "qux"), before.Contents,
		"before contents do not match")
	assert.Equal(t,
		text.Unlines(" foo", "baz", "qux"), after.Contents,
		"after contents do not match")

	// Checks that the line positions specified in
	assertLinesMatch := func(t *testing.T, v patchVersion) {
		// Number of line entries must match number of newlines.
		numLines := bytes.Count(v.Contents, []byte("\n"))
		if !assert.Len(t, v.Lines, numLines, "number of lines does not match") {
			return
		}

		for i, l := range v.Lines {
			wantLine := tillEOL(contents, fset.File(l.Pos).Offset(l.Pos))
			gotLine := tillEOL(v.Contents, l.Offset)

			assert.Equalf(t, string(wantLine), string(gotLine),
				"contents of line %d don't match", i)
		}
	}

	t.Run("before", func(t *testing.T) {
		assertLinesMatch(t, before)
	})

	t.Run("after", func(t *testing.T) {
		assertLinesMatch(t, after)
	})
}

// Returns the line in b starting at offset off.
func tillEOL(b []byte, off int) []byte {
	line := b[off:]
	if idx := bytes.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	return line
}
