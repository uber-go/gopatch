package section

import (
	"go/token"
	"testing"

	"github.com/uber-go/gopatch/internal/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	line := func(pos token.Pos, text string) *Line {
		return &Line{StartPos: pos, Text: []byte(text)}
	}

	type posInfo struct {
		L, C int // line and column number
	}

	tests := []struct {
		desc string
		give []byte

		want        Program
		wantPosInfo map[token.Pos]posInfo

		wantErrs []string
	}{
		{
			desc: "empty patch file",
			give: text.Unlines(),
			wantErrs: []string{
				"test.patch: unexpected EOF, at least one change is required",
			},
		},
		{
			desc: "bad header",
			give: text.Unlines(
				"@@ foo",
				"@@",
			),
			wantErrs: []string{
				`test.patch:1:1: unexpected "@@ foo", expected "@@" or "@ change_name @"`,
			},
		},
		{
			desc: "meta without body",
			give: text.Unlines(
				"@ foo @",
				"var x identifier",
			),
			wantErrs: []string{
				`test.patch:2:18: unexpected EOF, expected "@@"`,
			},
		},
		{
			desc: "empty change",
			give: text.Unlines(
				"@@",
				"@@",
			),
			want: Program{
				{
					HeaderPos: 1,
					AtPos:     4,
				},
			},
			wantPosInfo: map[token.Pos]posInfo{
				1: {L: 1, C: 1}, // @@
				3: {L: 1, C: 3},
				4: {L: 2, C: 1}, // @@
			},
		},
		{
			desc: "simple change",
			give: text.Unlines(
				"@@",
				"var x identifier",
				"@@",
				"-x()",
				"+x(42)",
			),
			want: Program{
				{
					HeaderPos: 1,
					Meta: Section{
						line(4, "var x identifier"),
					},
					AtPos: 21,
					Patch: Section{
						line(24, "-x()"),
						line(29, "+x(42)"),
					},
				},
			},
			wantPosInfo: map[token.Pos]posInfo{
				1:  {L: 1, C: 1}, // @@
				4:  {L: 2, C: 1}, // var x
				21: {L: 3, C: 1}, // @@
			},
		},
		{
			desc: "comments",
			give: text.Unlines(
				"# This patch adds an argument.",
				"@@",
				"  # x will match any identifier.",
				"var x identifier",
				"@@",
				"-x()",
				"# Given a call with no arguments...",
				"+x(42)",
				" # ... add 42 as an argument.",
			),
			want: Program{
				{
					HeaderPos: 32,
					Meta: Section{
						line(68, "var x identifier"),
					},
					AtPos: 85,
					Patch: Section{
						line(88, "-x()"),
						line(129, "+x(42)"),
					},
				},
			},
			wantPosInfo: map[token.Pos]posInfo{
				32:  {L: 2, C: 1}, // @@
				68:  {L: 4, C: 1}, // var x
				85:  {L: 5, C: 1}, // @@
				88:  {L: 6, C: 1}, // -x()
				129: {L: 8, C: 1}, // +x(42)
			},
		},
		{
			desc: "named change",
			give: text.Unlines(
				"@ cleanup @",
				"@@",
				"-foo(1, 2)",
				" bar(1, 2)",
				"+baz(1, 2)",
			),
			want: Program{
				{
					HeaderPos: 1,
					Name:      "cleanup",
					AtPos:     13,
					Patch: Section{
						line(16, "-foo(1, 2)"),
						line(27, " bar(1, 2)"),
						line(38, "+baz(1, 2)"),
					},
				},
			},
		},
		{
			desc: "named change compact",
			give: text.Unlines(
				"@foo@",
				"@@",
				"-foo()",
			),
			want: Program{
				{
					HeaderPos: 1,
					Name:      "foo",
					AtPos:     7,
					Patch: Section{
						line(10, "-foo()"),
					},
				},
			},
		},
		{
			desc: "multiple changes",
			give: text.Unlines(
				"@@",
				"var x identifier",
				"@@",
				"-x, err := foo(...)",
				"+err := foo(...)",
				"",
				"@ delete @",
				"var foo expression",
				"@@",
				"-foo()",
			),
			want: Program{
				{
					HeaderPos: 1,
					Meta: Section{
						line(4, "var x identifier"),
					},
					AtPos: 21,
					Patch: Section{
						line(24, "-x, err := foo(...)"),
						line(44, "+err := foo(...)"),
						line(61, ""),
					},
				},
				{
					HeaderPos: 62,
					Name:      "delete",
					Meta: Section{
						line(73, "var foo expression"),
					},
					AtPos: 92,
					Patch: Section{
						line(95, "-foo()"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fset := token.NewFileSet()

			got, err := Split(fset, "test.patch", tt.give)
			if len(tt.wantErrs) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrs {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err)

			require.NotEmpty(t, got, "must have at least one change")
			require.True(t, got[0].HeaderPos.IsValid(), "position must be valid")

			file := fset.File(got[0].HeaderPos)
			require.NotNil(t, file, "file must be added to FileSet")

			assert.Equal(t, tt.want, got, "changes did not match")

			for pos, info := range tt.wantPosInfo {
				gotPos := fset.Position(pos)
				assert.Equal(t, info.L, gotPos.Line, "line for %v did not match", pos)
				assert.Equal(t, info.C, gotPos.Column, "column for %v did not match", pos)
			}
		})
	}
}
