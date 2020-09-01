package engine

import (
	"reflect"
	"testing"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/stretchr/testify/assert"
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
