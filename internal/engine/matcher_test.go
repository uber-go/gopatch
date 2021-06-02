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

package engine

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/gopatch/internal/data"
)

type matchCase struct {
	desc string        // name of the test case
	give reflect.Value // value to match
	ok   bool          // expected result
}

// Runs the given matcher on the provided test cases in-order, threading the
// data.Data between them. Returns the final data.Data and whether all cases
// matched or not.
//
// If all cases did not match, the test has already been marked as
// unsuccessful.
func assertMatchCases(t *testing.T, m Matcher, d data.Data, tests []matchCase) (data.Data, bool) {
	valid := true
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			newd, ok := m.Match(tt.give, d, Region{})
			if ok {
				// Carry data over in case of successful matches.
				d = newd
			}
			valid = assert.Equal(t, tt.ok, ok) && valid
		})
	}
	return d, valid
}
