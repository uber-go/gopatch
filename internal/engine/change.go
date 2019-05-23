package engine

import "github.com/uber-go/gopatch/internal/parse"

// Change is a single Change in a program.
type Change struct {
	Name string
	Meta *Meta
}

func (c *compiler) compileChange(achange *parse.Change) *Change {
	// TODO(abg): Compile the patch for this change.

	return &Change{
		Name: achange.Name, // TODO(abg): validate name
		Meta: c.compileMeta(achange.Meta),
	}
}
