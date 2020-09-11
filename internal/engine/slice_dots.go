package engine

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
)

// SliceDotsMatcher implements support for "..." in portions of the AST where
// slices of values are expected.
type SliceDotsMatcher struct {
	// TODO: Type

	// List of contiguous sections to match against.
	Sections [][]Matcher // inv: len > 0

	// Positions at which dots were found.
	Dots []token.Pos // inv: len(dots) = len(sections) - 1
}

func (c *matcherCompiler) compileSliceDots(items reflect.Value, isDots func(ast.Node) bool) Matcher {
	// TODO(abg): Validate that this is called with ast.Node-compatible types
	// only.

	var (
		sections [][]Matcher
		current  []Matcher
		dots     []token.Pos
	)
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i)
		if n, ok := item.Interface().(ast.Node); ok && isDots(n) {
			dotPos := n.Pos()
			c.dots = append(c.dots, dotPos)
			dots = append(dots, dotPos)
			sections = append(sections, current)
			current = nil
		} else {
			current = append(current, c.compile(item))
		}
	}
	sections = append(sections, current)

	// Optimization: If there are no "..."s, we can use the faster slice
	// matcher.
	if len(sections) == 1 {
		return SliceMatcher{Items: sections[0]}
	}

	return SliceDotsMatcher{Sections: sections, Dots: dots}
}

// Match matches
func (m SliceDotsMatcher) Match(got reflect.Value, d data.Data, r Region) (data.Data, bool) {
	gotItems := make([]reflect.Value, got.Len())
	for i := 0; i < got.Len(); i++ {
		gotItems[i] = got.Index(i)
	}

	// The first section must match in-place.
	idx, d, ok := matchPrefix(m.Sections[0], gotItems, d, sectionRegion(gotItems, r, 0, len(m.Sections[0])), 0)
	if !ok {
		return d, false
	}

	for i, section := range m.Sections[1:] {
		idx, d, ok = findSection(m.Dots[i], section, gotItems, d, r, idx)
		if !ok {
			return d, false
		}
	}

	return d, idx == len(gotItems)
}

// Returns Region for items[start:end].
func sectionRegion(items []reflect.Value, r Region, start, end int) Region {
	if start > 0 {
		// If this isn't the first item, use the end position of the previous
		// sibling as the start of the region.
		r.Pos = items[start-1].Interface().(ast.Node).End()
	}
	if end < len(items) {
		// If this isn't the last item, use the start position of the next
		// sibling as the end of this region.
		r.End = items[end].Interface().(ast.Node).Pos()
	}
	return r
}

// matchPrefix matches all items in want starting at got[idx]. If all items
// were matched, the new index for the remaining matches is returned.
func matchPrefix(want []Matcher, got []reflect.Value, d data.Data, r Region, idx int) (newIdx int, _ data.Data, ok bool) {
	if len(want) > len(got)-idx {
		return idx, d, false
	}

	for i, m := range want {
		var ok bool
		d, ok = m.Match(got[i+idx], d, r)
		if !ok {
			return idx, d, false
		}
	}

	return idx + len(want), d, true
}

// findSection attempts to match want starting at got[idx], moving onto idx+1,
// idx+2, and so on until a match is found. Returns the new index for the
// remaining matches.
//
// Invariant: If ok is true, a list of skipped items will have been pushed to
// Data.
func findSection(dots token.Pos, want []Matcher, got []reflect.Value, d data.Data, r Region, idx int) (newIdx int, _ data.Data, ok bool) {
	// Special case: Looking for "..." at the end of the list. Skip everything
	// in got.
	if len(want) == 0 {
		r := sectionRegion(got, r, idx, len(got))
		d := pushSliceDotsSkipped(d, dots, got[idx:], r)
		return matchPrefix(want, got, d, r, len(got))
	}

	for i := idx; i < len(got); i++ {
		r := sectionRegion(got, r, idx, i)
		newIdx, newD, ok := matchPrefix(want, got, pushSliceDotsSkipped(d, dots, got[idx:i], r), r, i)
		if ok {
			return newIdx, newD, ok
		}
	}

	return idx, d, false
}

// SliceDotsReplacer replaces target nodes and reproduces the values captured by
// "..." in places which the AST expects a slice.
type SliceDotsReplacer struct {
	Type     reflect.Type
	Sections [][]Replacer // inv: len > 0

	// Positions at which dots were found.
	Dots []token.Pos // inv: len(dots) = len(sections) - 1

	dotAssoc map[token.Pos]token.Pos
}

func (c *replacerCompiler) compileSliceDots(items reflect.Value, isDots func(ast.Node) bool) Replacer {
	if items.IsNil() {
		return ZeroReplacer{Type: items.Type()}
	}

	var (
		sections [][]Replacer
		current  []Replacer
		dots     []token.Pos
	)
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i)
		if n, ok := item.Interface().(ast.Node); ok && isDots(n) {
			dotPos := n.Pos()
			c.dots = append(c.dots, dotPos)
			dots = append(dots, dotPos)
			sections = append(sections, current)
			current = nil
		} else {
			current = append(current, c.compile(item))
		}
	}
	sections = append(sections, current)

	// Optimization: If there are no "..."s, we can use the faster slice
	// replacer.
	if len(sections) == 1 {
		return SliceReplacer{Type: items.Type(), Items: sections[0]}
	}

	// TODO: we probably just need some sort of "what's my associated
	// dot pos/ID" functionality injected here instead of
	// grabbing the map from compiler.
	return SliceDotsReplacer{
		Type:     items.Type(),
		Dots:     dots,
		Sections: sections,
		dotAssoc: c.dotAssoc,
	}
}

// Replace replaces target Nodes in slices where elements may have been elided
// in the patch.
func (r SliceDotsReplacer) Replace(d data.Data, cl Changelog, pos token.Pos) (reflect.Value, error) {
	// TODO: Recurse into Cursor

	type skippedSection struct {
		Items  []reflect.Value
		Region Region
	}

	var skipped []skippedSection
	for _, dotPos := range r.Dots {
		items, region := lookupSliceDotsSkipped(d, r.dotAssoc[dotPos])
		skipped = append(skipped, skippedSection{
			Items:  items,
			Region: region,
		})
	}

	var items []reflect.Value
	for i, section := range r.Sections {
		for _, replacer := range section {
			item, err := replacer.Replace(d, cl, pos)
			if err != nil {
				return reflect.Value{}, err
			}
			items = append(items, item)
		}

		if i < len(skipped) {
			s := skipped[i]
			items = append(items, s.Items...)
			cl.Unchanged(s.Region.Pos, s.Region.End)
			pos = s.Region.End
		}
	}

	// TODO: Need to explicitly handle nil vs empty
	if len(items) == 0 {
		return reflect.Zero(r.Type), nil
	}

	result := reflect.MakeSlice(r.Type, len(items), len(items))
	for i, item := range items {
		result.Index(i).Set(item)
	}
	return result, nil
}

type sliceDotsKey token.Pos

type sliceDotsData struct {
	Skipped []reflect.Value
	Region  Region
}

// Pushes the list of skipped items to Data. Skipped items are got[start:end].
func pushSliceDotsSkipped(d data.Data, dots token.Pos, got []reflect.Value, r Region) data.Data {
	return data.WithValue(d, sliceDotsKey(dots), sliceDotsData{Skipped: got, Region: r})
}

func lookupSliceDotsSkipped(d data.Data, dots token.Pos) (result []reflect.Value, region Region) {
	var sd sliceDotsData
	// TODO(abg): If ok is false, we couldn't find data for this "...". That's
	// invalid and we should report that.
	_ = data.Lookup(d, sliceDotsKey(dots), &sd)
	return sd.Skipped, sd.Region
}
