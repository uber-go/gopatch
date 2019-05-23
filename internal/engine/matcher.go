package engine

import "reflect"

// Matcher matches values in a Go AST. It is built from the "-" portion of a
// patch.
type Matcher interface {
	// Match reports whether the provided value from a Go AST matches it.
	Match(reflect.Value) (ok bool)
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

func (f matcherFunc) Match(v reflect.Value) bool {
	return f(v)
}

// nilMatcher is a Matcher that only matches nil values.
var nilMatcher Matcher = matcherFunc(func(got reflect.Value) bool { return got.IsNil() })
