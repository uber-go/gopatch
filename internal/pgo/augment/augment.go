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
