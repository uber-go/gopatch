package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	d := New()
	assert.Empty(t, d.Keys())
	assert.Nil(t, d.Value(42))
}

func TestValues(t *testing.T) {
	t.Run("indexed", func(t *testing.T) {
		testValues(t, Index)
	})

	t.Run("no index", func(t *testing.T) {
		testValues(t, func(d Data) Data { return d })
	})

	t.Run("indexed twice", func(t *testing.T) {
		testValues(t, func(d Data) Data {
			return Index(Index(d))
		})
	})
}

func testValues(t *testing.T, maybeIndex func(Data) Data) {
	type key string

	t.Run("adding a value", func(t *testing.T) {
		d := New()
		d = WithValue(d, key("foo"), 1)
		d = WithValue(d, key("bar"), 2)
		d = maybeIndex(d)

		t.Run("Keys", func(t *testing.T) {
			assert.ElementsMatch(t, []key{"foo", "bar"}, d.Keys())
		})

		t.Run("Value/match", func(t *testing.T) {
			assert.Equal(t, 1, d.Value(key("foo")))
			assert.Equal(t, 2, d.Value(key("bar")))
		})

		t.Run("Value/wrong type", func(t *testing.T) {
			assert.Nil(t, d.Value("foo"))
		})
	})

	t.Run("adding and ignoring", func(t *testing.T) {
		d1 := WithValue(New(), key("foo"), 1)
		d2 := WithValue(maybeIndex(d1), key("bar"), 2)
		d3 := maybeIndex(WithValue(d1, key("baz"), 3))

		t.Run("Keys", func(t *testing.T) {
			assert.ElementsMatch(t, []key{"foo"}, d1.Keys())
			assert.ElementsMatch(t, []key{"foo", "bar"}, d2.Keys())
			assert.ElementsMatch(t, []key{"foo", "baz"}, d3.Keys())
		})

		t.Run("Value/foo", func(t *testing.T) {
			assert.Equal(t, 1, d1.Value(key("foo")))
			assert.Equal(t, 1, d2.Value(key("foo")))
			assert.Equal(t, 1, d3.Value(key("foo")))
		})

		t.Run("Value/bar", func(t *testing.T) {
			assert.Nil(t, d1.Value(key("bar")))
			assert.Equal(t, 2, d2.Value(key("bar")))
			assert.Nil(t, d3.Value(key("bar")))
		})

		t.Run("Value/baz", func(t *testing.T) {
			assert.Nil(t, d1.Value(key("baz")))
			assert.Nil(t, d2.Value(key("baz")))
			assert.Equal(t, 3, d3.Value(key("baz")))
		})
	})
}
