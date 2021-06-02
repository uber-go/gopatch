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
	"bytes"
	"go/token"
	"sort"
)

// LinePos contains positional information about a line in a buffer.
type LinePos struct {
	// Offset of the first character of this line.
	Offset int

	// Original position from which this line was extracted.
	Pos token.Pos
}

// ToBytes converts a Section to its raw byte contents. A sorted list mapping
// offsets in the returned byte slice to original token.Pos values is included
// in the response.
func ToBytes(s Section) (src []byte, lines []LinePos) {
	var buff bytes.Buffer
	for _, line := range s {
		lines = append(lines, LinePos{
			Offset: buff.Len(),
			Pos:    line.Pos(),
		})
		buff.Write(line.Text)
		buff.WriteByte('\n')
	}
	sort.Sort(byOffset(lines))
	return buff.Bytes(), lines
}

type byOffset []LinePos

func (ls byOffset) Len() int { return len(ls) }

func (ls byOffset) Less(i int, j int) bool {
	return ls[i].Offset < ls[j].Offset
}

func (ls byOffset) Swap(i int, j int) {
	ls[i], ls[j] = ls[j], ls[i]
}
