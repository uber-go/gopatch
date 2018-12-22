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
