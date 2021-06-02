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

package augment

import (
	"bytes"
	"fmt"
	"sort"
)

const (
	_fakePackage = "package _"
	_fakeFunc    = "func _() "
)

// PosAdjustment specifies how much token.Pos's have to be adjusted after
// different offsets. This is required because to make the pgo source valid Go
// code, we may have to geneate additional code around it. This will mess up
// positioning information so we track how much we're affecting the positions
// and starting at what offsets.
type PosAdjustment struct {
	Offset   int
	ReduceBy int
}

// rewrites the source to be valid Go syntax. Offsets in the augmentations are
// updated to reflect offsets in new []byte.
func rewrite(src []byte, augs []Augmentation) ([]byte, []PosAdjustment) {
	var (
		pos         int
		dst         bytes.Buffer
		tail        bytes.Buffer
		adjustments []PosAdjustment
		reduceBy    int
	)
	// augs will always be in the format,
	//
	//   FakePackage?, FakeFunc?, Other*
	//
	// That is, FakePackage and/or FakeFunc MAY be present at the front of the
	// list in that order. This tells us to generate the fake package and func
	// clauses. Other augmentations MAY be present after that.
	//
	// Sort all augs by Start offset to retain the above ordering while ensuring
	// that augmentations get written to the `dst` Buffer in order.
	sort.Slice(augs, func(i, j int) bool { return augs[i].Start() < augs[j].Start() })
	for _, aug := range augs {
		start, end := aug.Start(), aug.End()
		dst.Write(src[pos:start])

		switch a := aug.(type) {
		case *FakePackage:
			a.PackageStart = dst.Len()
			fmt.Fprintln(&dst, _fakePackage)
			reduceBy += dst.Len() - a.PackageStart
			adjustments = append(adjustments, PosAdjustment{
				Offset:   a.PackageStart,
				ReduceBy: reduceBy,
			})
		case *FakeFunc:
			a.FuncStart = dst.Len()
			dst.WriteString(_fakeFunc)
			if a.Braces {
				dst.WriteString("{\n")
				tail.WriteString("}\n")
			}
			reduceBy += dst.Len() - a.FuncStart
			adjustments = append(adjustments, PosAdjustment{
				Offset:   a.FuncStart,
				ReduceBy: reduceBy,
			})
		case *Dots:
			a.DotsStart = dst.Len()
			if a.Named {
				// no PosAdjustment: len(_ d) == len(...)
				dst.WriteString("_ d")
			} else {
				// no PosAdjustment: len(dts) == len(...)
				dst.WriteString("dts")
			}
			a.DotsEnd = dst.Len()
		default:
			panic(fmt.Sprintf("unknown augmentation type %T", a))
		}
		pos = end
	}
	dst.Write(src[pos:])
	dst.Write(tail.Bytes())
	return dst.Bytes(), adjustments
}
