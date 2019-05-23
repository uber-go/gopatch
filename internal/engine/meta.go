package engine

import (
	"go/token"

	"github.com/uber-go/gopatch/internal/parse"
)

// MetavarType defines the different types of metavariables accepted in a
// '@@' section.
type MetavarType int

// Supported metavariable types.
const (
	ExprMetavarType  MetavarType = iota + 1 // expression
	IdentMetavarType                        // identifier
)

// Meta is the compiled representaton of a Meta section.
type Meta struct {
	// Variables defined in this Meta section and their types.
	Vars map[string]MetavarType
}

// LookupVar returns the type of the given metavariable or zero value if it
// wasn't found.
func (m *Meta) LookupVar(name string) MetavarType {
	if m == nil {
		return 0
	}
	return m.Vars[name]
}

func (c *compiler) compileMeta(m *parse.Meta) *Meta {
	vars := make(map[string]MetavarType)
	declPos := make(map[string]token.Pos)

	for _, decl := range m.Vars {
		var t MetavarType
		switch decl.Type.Name {
		case "identifier":
			t = IdentMetavarType
		case "expression":
			t = ExprMetavarType
		default:
			c.errf(decl.Type.Pos(), "unknown metavariable type %q", decl.Type.Name)
			continue
		}

		for _, name := range decl.Names {
			if name.Name == "_" {
				// Underscore isn't a variable declaration.
				continue
			}

			if pos, conflict := declPos[name.Name]; conflict {
				c.errf(name.Pos(), "cannot define metavariable %q: "+
					"name already taken by metavariable defined at %v", name.Name,
					c.fset.Position(pos))
				continue
			}
			vars[name.Name] = t
			declPos[name.Name] = name.Pos()
		}
	}

	return &Meta{Vars: vars}
}
