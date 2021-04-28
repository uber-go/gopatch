package section

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/gopatch/internal/text"
)

func TestToBytes(t *testing.T) {
	tests := []struct {
		desc      string
		give      Section
		wantSrc   []byte
		wantLines []LinePos
	}{
		{
			desc: "simple",
			give: Section{
				{StartPos: 1, Text: []byte("foo")},
				{StartPos: 5, Text: []byte("bar")},
			},
			wantSrc: text.Unlines(
				"foo",
				"bar",
			),
			wantLines: []LinePos{
				{Offset: 0, Pos: 1},
				{Offset: 4, Pos: 5},
			},
		},
		{
			desc: "missing portions",
			give: Section{
				{StartPos: 1, Text: []byte("foo")},
				{StartPos: 20, Text: []byte("bar")},
				{StartPos: 30, Text: []byte("baz")},
			},
			wantSrc: text.Unlines(
				"foo",
				"bar",
				"baz",
			),
			wantLines: []LinePos{
				{Offset: 0, Pos: 1},
				{Offset: 4, Pos: 20},
				{Offset: 8, Pos: 30},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			gotSrc, gotLines := ToBytes(tt.give)
			assert.Equal(t, tt.wantSrc, gotSrc)
			assert.Equal(t, tt.wantLines, gotLines)
		})
	}
}
