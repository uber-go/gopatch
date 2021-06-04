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

// This currently only has unit-tests for testing verboseLogger.
// Refer to e2e_test.go for some integration tests.

package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerboseLoggerOn(t *testing.T) {
	var b bytes.Buffer
	logger := verboseLogger{&b, true}

	logger.Log("Hello")
	logger.Log("Hi there")
	assert.Equal(t, "[gopatch] Hello\n[gopatch] Hi there\n", b.String())
}

func TestVerboseLoggerOff(t *testing.T) {
	var b bytes.Buffer
	logger := verboseLogger{&b, false}

	logger.Log("Hello")
	logger.Log("Hi there")
	assert.Equal(t, 0, b.Len())
}
