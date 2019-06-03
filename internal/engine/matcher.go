package engine

import (
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
)

// Matcher matches values in a Go AST. It is built from the "-" portion of a
// patch.
type Matcher interface {
	// Match is called with the a value from the AST being compared with the
	// match data captured so far.
	//
	// Match reports whether the match succeeded and if so, returns the
	// original or a different Data object containing additional match data.
	Match(reflect.Value, data.Data) (_ data.Data, ok bool)
}

// matcherCompiler compiles the "-" portion of a patch into a Matcher which
// will report whether another Go AST matches it.
type matcherCompiler struct {
}

func newMatcherCompiler() *matcherCompiler {
	return &matcherCompiler{}
}

func (c *matcherCompiler) compile(v reflect.Value) Matcher {
	// NOTE: We're currently relying entirely on the reflection-based matcher.
	// Upcoming changes will flesh out this method further.
	return c.compileGeneric(v)
}

type matcherFunc func(reflect.Value) bool

func (f matcherFunc) Match(v reflect.Value, d data.Data) (data.Data, bool) {
	return d, f(v)
}

// nilMatcher is a Matcher that only matches nil values.
var nilMatcher Matcher = matcherFunc(func(got reflect.Value) bool { return got.IsNil() })
