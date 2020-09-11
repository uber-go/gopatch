// Package augment facilitates adding syntax to the Go AST.
//
// It works by replacing invalid syntax with valid code, recording these
// positions as it does so, allowing us to later transform the parsed AST for
// affected nodes.
package augment

// Augmentation is an addition to the Go syntax for pgo.
type Augmentation interface {
	augmentation()

	// Start returns the offset at which the augmentation was found.
	Start() int

	// End returns the offset of the first character immediately after this
	// augmentation.
	//
	// Given an augmentation a in text, text[a.Start():a.End()] fully
	// encapsulates the text of the augmentation.
	End() int
}

// Dots is a "..." operator appearing in the Go source outside of places
// allowed by the Go syntax.
type Dots struct {
	DotsStart, DotsEnd int

	// Named indicates whether the dots replace a named entity — such
	// as the named arguments or results of a function.
	Named bool
}

func (*Dots) augmentation() {}

// Start offset of Dots.
func (d *Dots) Start() int { return d.DotsStart }

// End offset of Dots.
func (d *Dots) End() int { return d.DotsEnd }

// LDots is a "<..." operator.
type LDots struct {
	LDotsStart, LDotsEnd int
}

func (*LDots) augmentation() {}

// Start offset of LDots.
func (d *LDots) Start() int { return d.LDotsStart }

// End offset of LDots.
func (d *LDots) End() int { return d.LDotsEnd }

// RDots is a "...>" operator.
type RDots struct {
	RDotsStart, RDotsEnd int
}

func (*RDots) augmentation() {}

// Start offset of RDots.
func (d *RDots) Start() int { return d.RDotsStart }

// End offset of RDots.
func (d *RDots) End() int { return d.RDotsEnd }

// FakePackage is a fake package clause included in the code. This is needed
// if the source didn't contain a package clause.
type FakePackage struct {
	PackageStart int // position of "package" keyword
}

func (*FakePackage) augmentation() {}

// Start offset for FakePackage.
func (p *FakePackage) Start() int { return p.PackageStart }

// End offset for FakePackage. This is meaningless for FakePackage.
func (p *FakePackage) End() int { return p.PackageStart }

// FakeFunc is a fake function block generated in the code. This is needed if
// the source didn't open with a valid top-level declaration.
type FakeFunc struct {
	FuncStart int  // position of "func" keyword
	Braces    bool // whether a { ... } was added
}

func (*FakeFunc) augmentation() {}

// Start offset for FakeFunc.
func (f *FakeFunc) Start() int { return f.FuncStart }

// End offset for FakeFunc. This is meaningless for FakeFunc.
func (f *FakeFunc) End() int { return f.FuncStart }

// Augment takes the provided pgo syntax and returns valid Go syntax, a list
// of augmentations made to it, and position adjustments required in the
// parsed AST.
func Augment(src []byte) ([]byte, []Augmentation, []PosAdjustment, error) {
	augs, err := find(src)
	if err != nil {
		return nil, nil, nil, err
	}
	src, adjs := rewrite(src, augs)
	return src, augs, adjs, nil
}
