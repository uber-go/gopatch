package engine

import (
	"errors"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
)

// TODO(abg): This has a fair amount of logic similar to stmt_list's
// reproduction logic. We can probably generalize this.

// ForDotsMatcher represents a "for ..." stmt, matching both, for and range
// statements.
type ForDotsMatcher struct {
	Dots token.Pos
	Body Matcher
}

func (c *matcherCompiler) compileForStmt(v reflect.Value) Matcher {
	stmt := v.Interface().(*ast.ForStmt)
	if _, isDots := stmt.Cond.(*pgo.Dots); !isDots || stmt.Init != nil || stmt.Post != nil {
		// Not a "for ...". Fall back to the usual logic.
		return c.compileGeneric(v)
	}
	dotPos := stmt.Cond.Pos()
	c.dots = append(c.dots, dotPos)
	return ForDotsMatcher{
		Dots: dotPos,
		Body: c.compile(reflect.ValueOf(stmt.Body)),
	}
}

// Match matches either a for statement or a range statement.
func (m ForDotsMatcher) Match(got reflect.Value, d data.Data, r Region) (data.Data, bool) {
	var (
		bodyField       forDotsField
		bodyFieldRegion Region
		otherFields     []forDotsField
	)

	var t reflect.Type
	switch got.Type() {
	case goast.ForStmtPtrType, goast.RangeStmtPtrType:
		got = got.Elem()
		t = got.Type()
	default:
		return d, false
	}

	for i := 0; i < t.NumField(); i++ {
		f := forDotsField{Idx: i, Value: got.Field(i)}
		// The BlockStmt is named Body on both types.
		if t.Field(i).Name == "Body" {
			bodyField = f
			body := f.Value.Interface().(ast.Node)

			// r tracks the unchanged region. In this case, it
			// ends when the body starts.
			r.End = body.Pos()
			bodyFieldRegion = nodeRegion(body)
		} else {
			otherFields = append(otherFields, f)
		}
	}

	d = data.WithValue(
		d,
		forDotsKey(m.Dots),
		forDotsData{
			Type:         t,
			BodyFieldIdx: bodyField.Idx,
			OtherFields:  otherFields,
			Region:       r,
		},
	)

	return m.Body.Match(bodyField.Value, d, bodyFieldRegion)
}

// ForDotsReplacer replaces a "for ...".
type ForDotsReplacer struct {
	Dots token.Pos
	Body Replacer

	dotAssoc map[token.Pos]token.Pos
}

func (c *replacerCompiler) compileForStmt(v reflect.Value) Replacer {
	stmt := v.Interface().(*ast.ForStmt)
	if _, isDots := stmt.Cond.(*pgo.Dots); !isDots || stmt.Init != nil || stmt.Post != nil {
		// Not a "for ...". Fall back to the usual logic.
		return c.compileGeneric(v)
	}
	dotPos := stmt.Cond.Pos()
	c.dots = append(c.dots, dotPos)
	return ForDotsReplacer{
		Dots:     dotPos,
		Body:     c.compile(reflect.ValueOf(stmt.Body)),
		dotAssoc: c.dotAssoc,
	}
}

// Replace rebuilds a For or Range statement from the originally captured
// fields.
func (r ForDotsReplacer) Replace(d data.Data, cl Changelog, pos token.Pos) (reflect.Value, error) {
	var fd forDotsData
	if !data.Lookup(d, forDotsKey(r.dotAssoc[r.Dots]), &fd) {
		return reflect.Value{}, errors.New("match data not found for 'for ...': " +
			"are you sure that the line appears on a context line without a preceding '-' or '+'?")
	}

	cl.Unchanged(fd.Region.Pos, fd.Region.End)

	stmt := reflect.New(fd.Type).Elem()

	// Reproduce fields besides the body as-is.
	for _, f := range fd.OtherFields {
		stmt.Field(f.Idx).Set(f.Value)
	}

	body, err := r.Body.Replace(d, cl, pos)
	if err != nil {
		return reflect.Value{}, err
	}

	stmt.Field(fd.BodyFieldIdx).Set(body)
	return stmt.Addr(), nil
}

type forDotsKey token.Pos

type forDotsField struct {
	Idx   int           // index of the field in Type
	Value reflect.Value // captured value of the field
}

type forDotsData struct {
	// Type of statement that we matched.
	//
	// This is one of ForStmt or RangeStmt.
	Type reflect.Type

	// Field index in Type at which the "Body *ast.BlockStmt" field is
	// present.
	BodyFieldIdx int

	OtherFields []forDotsField

	Region Region
}
