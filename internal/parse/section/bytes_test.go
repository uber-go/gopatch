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
