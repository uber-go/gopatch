package engine

import (
	"fmt"
	"go/token"

	"github.com/google/go-intervals/intervalset"
)

// Changelog records the ranges of positions changed over the course of
// successive patches.
type Changelog struct {
	// we maintain separate plus and minus sets because we want to be able to
	// submit changed/unchanged requests out of order.
	plus  *intervalset.Set
	minus *intervalset.Set
}

// NewChangelog builds a new, empty Changelog
func NewChangelog() Changelog {
	return Changelog{
		plus:  intervalset.Empty(),
		minus: intervalset.Empty(),
	}
}

// Changed records the portion of code altered in this match to the Changelog.
func (c Changelog) Changed(start, end token.Pos) {
	c.plus.Add((&span{Start: start, End: end}).AsSet())
}

// Unchanged records the portion of unaltered code to the Changelog.
func (c Changelog) Unchanged(start, end token.Pos) {
	c.minus.Add((&span{Start: start, End: end}).AsSet())
}

// Interval represents a consecutive set of positions in the source file.
type Interval struct {
	Start, End token.Pos
}

// ChangedIntervals returns the sum of the Intervals recorded to the Changelog.
func (c Changelog) ChangedIntervals() []Interval {
	set := intervalset.Empty()
	set.Add(c.plus)
	set.Sub(c.minus)

	var ivals []Interval
	set.Intervals(func(i intervalset.Interval) bool {
		s := i.(*span)
		ivals = append(ivals, Interval{
			Start: s.Start,
			End:   s.End,
		})
		return true
	})
	return ivals
}

type span struct{ Start, End token.Pos }

func minPos(l, r token.Pos) token.Pos {
	if l < r {
		return l
	}
	return r
}

func maxPos(l, r token.Pos) token.Pos {
	if l > r {
		return l
	}
	return r
}

func zeroSpan() *span { return &span{} }

func validOrZero(s *span) *span {
	if s.Start < s.End {
		return s
	}
	return zeroSpan()
}

var _ intervalset.Interval = (*span)(nil)

func (s *span) String() string {
	return fmt.Sprintf("[%v, %v)", s.Start, s.End)
}

func (s *span) Intersect(other intervalset.Interval) intervalset.Interval {
	i := other.(*span)
	return validOrZero(&span{
		Start: maxPos(s.Start, i.Start),
		End:   minPos(s.End, i.End),
	})
}

func (s *span) Before(i intervalset.Interval) bool {
	if i == nil {
		// XXX: intervalset passes in nil when an interval is out of range (I
		// think?). There isn't a lot of visibility; it runs stuff in separate
		// goroutines for some reason.
		return true
	}
	return s.End <= i.(*span).Start
}

func (s *span) IsZero() bool {
	return s.Start == 0 && s.End == 0
}

func (s *span) Bisect(i intervalset.Interval) (intervalset.Interval, intervalset.Interval) {
	intersection := s.Intersect(i).(*span)
	if intersection.IsZero() {
		if s.Before(i) {
			return s, zeroSpan()
		}
		return zeroSpan(), s
	}

	return validOrZero(
			&span{
				Start: s.Start,
				End:   intersection.Start,
			}), validOrZero(&span{
			Start: intersection.End,
			End:   s.End,
		})
}

func (s *span) Adjoin(other intervalset.Interval) intervalset.Interval {
	i := other.(*span)
	if s.End == i.Start {
		return &span{Start: s.Start, End: i.End}
	}
	if i.End == s.Start {
		return &span{Start: i.Start, End: s.End}
	}
	return zeroSpan()
}

func (s *span) Encompass(other intervalset.Interval) intervalset.Interval {
	i := other.(*span)
	return &span{
		Start: minPos(i.Start, s.Start),
		End:   maxPos(i.End, s.End),
	}
}

func (s *span) AsSet() *intervalset.ImmutableSet {
	return intervalset.NewImmutableSet([]intervalset.Interval{s})
}
