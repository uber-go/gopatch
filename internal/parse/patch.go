package parse

import (
	"bytes"
	"io"

	"github.com/uber-go/gopatch/internal/parse/section"
)

// parsePatch parses a Patch from the given source.
func (p *parser) parsePatch(i int, c *section.Change) (*Patch, error) {
	if len(c.Patch) == 0 {
		return nil, p.errf(c.AtPos, "invalid change: patch cannot be empty")
	}

	patch := Patch{
		StartPos: c.Patch[0].Pos(),
		EndPos:   c.End(),
	}

	splitPatch(c.Patch)
	// TODO(abg): Parse the two versions.

	return &patch, nil
}

// patchVersion is one of the versions of a patch specified in a unified diff.
type patchVersion struct {
	// Contents of the file.
	Contents []byte

	// Positional information for each line in Contents.
	//
	// Each LinePos contains matches an offset in Contents to a token.Pos in
	// the original patch file.
	Lines []section.LinePos
}

// splitPatch splits a patch into the before and after versions of the
// file.
//
// Given the unified diff,
//
//   foo
//  -bar
//  +baz
//   qux
//
// This functions splits it into,
//
//  Before  After
//  ------  -----
//  foo     foo
//  bar     baz
//  qux     qux
func splitPatch(patch section.Section) (before, after patchVersion) {
	var (
		minus, plus bytes.Buffer
		both        = io.MultiWriter(&minus, &plus)

		minusLines, plusLines []section.LinePos

		newline = []byte("\n")
	)

	for _, line := range patch {
		// If true, the corresponding item won't get a LinePos entry because
		// it wasn't written to this time.
		var skipMinus, skipPlus bool

		w := both
		if len(line.Text) > 0 {
			switch line.Text[0] {
			case '-':
				skipPlus = true
				w = &minus
				line.Text = line.Text[1:]
				line.StartPos++ // '-'

			case '+':
				skipMinus = true
				w = &plus
				line.Text = line.Text[1:]
				line.StartPos++ // '+'
			}
		}

		if !skipMinus {
			minusLines = append(minusLines, section.LinePos{
				Offset: minus.Len(),
				Pos:    line.StartPos,
			})
		}

		if !skipPlus {
			plusLines = append(plusLines, section.LinePos{
				Offset: plus.Len(),
				Pos:    line.StartPos,
			})
		}

		w.Write(line.Text)
		w.Write(newline)
	}

	return patchVersion{
			Contents: minus.Bytes(),
			Lines:    minusLines,
		}, patchVersion{
			Contents: plus.Bytes(),
			Lines:    plusLines,
		}
}
