package astdiff

import (
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/goast"
)

type value struct {
	t reflect.Type

	// Only one of the following three is set.
	isNil    bool
	value    interface{}
	Elem     *value
	Children []*value

	// Set only if this is a Node.
	IsNode   bool
	pos, end token.Pos
	Comments []*ast.CommentGroup
}

func (v *value) Type() reflect.Type     { return v.t }
func (v *value) Kind() reflect.Kind     { return v.t.Kind() }
func (v *value) Pos() token.Pos         { return v.pos }
func (v *value) End() token.Pos         { return v.end }
func (v *value) Interface() interface{} { return v.value }
func (v *value) Len() int               { return len(v.Children) }
func (v *value) IsNil() bool            { return v.isNil }

func snapshot(v reflect.Value, cmap ast.CommentMap) (val *value) {
	t := v.Type()

	switch t {
	case goast.ObjectPtrType:
		// Ident.obj forms a cycle.
		return &value{t: t, isNil: true}
	case goast.CommentGroupPtrType, goast.ScopePtrType:
		// Snapshots don't care about comment groups.
		return &value{t: t, isNil: true}
	case goast.PosType:
		return &value{
			t:     t,
			value: v.Interface(),
		}
	}

	if t.Implements(goast.NodeType) && !v.IsNil() {
		defer func(n ast.Node) {
			val.IsNode = true
			val.pos = n.Pos()
			val.end = n.End()
			val.Comments = cmap[n]
		}(v.Interface().(ast.Node))
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return &value{t: t, isNil: true}
		}

		return &value{
			t:    t,
			Elem: snapshot(v.Elem(), cmap),
		}
	case reflect.Slice:
		children := make([]*value, v.Len())
		for i := 0; i < v.Len(); i++ {
			children[i] = snapshot(v.Index(i), cmap)
		}
		return &value{
			t:        t,
			Children: children,
		}

	case reflect.Struct:
		children := make([]*value, v.NumField())
		for i := 0; i < v.NumField(); i++ {
			children[i] = snapshot(v.Field(i), cmap)
		}
		return &value{
			t:        t,
			Children: children,
		}

	default:
		return &value{
			t:     t,
			value: v.Interface(),
		}
	}
}

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
