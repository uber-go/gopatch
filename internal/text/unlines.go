package text

import (
	"bytes"
	"fmt"
)

// Unlines takes a list of strings and joins them with newlines between them,
// including a trailing newline at the end.
func Unlines(lines ...string) []byte {
	var out bytes.Buffer
	for _, l := range lines {
		fmt.Fprintln(&out, l)
	}
	return out.Bytes()
}
