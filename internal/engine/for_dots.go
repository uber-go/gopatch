package engine

import (
	"errors"
	"go/ast"
	"reflect"

	"github.com/uber-go/gopatch/internal/data"
	"github.com/uber-go/gopatch/internal/goast"
	"github.com/uber-go/gopatch/internal/pgo"
)

// TODO(abg): This has a fair amount of logic similar to stmt_list's
// reproduction logic. We can probably generalize this.

// ForDotsMatcher represents a "for ..." stmt, matching both, for and range
// statements.
//
// The position of the "..." must be the same between the matcher and the
// compiler. This means that the "for ..." must occur on a context line, not
// one that has a leading "-" or "+'.
type ForDotsMatcher struct {
	DotsLine, DotsColumn int // line and column on which dots appear
	Body                 Matcher
}

func (c *matcherCompiler) compileForStmt(v reflect.Value) Matcher {
	stmt := v.Interface().(*ast.ForStmt)

	if _, isDots := stmt.Cond.(*pgo.Dots); !isDots || stmt.Init != nil || stmt.Post != nil {
		// Not a "for ...". Fall back to the usual logic.
		return c.compileGeneric(v)
	}

	dpos := c.fset.Position(stmt.Cond.Pos())
	return ForDotsMatcher{
		DotsLine:   dpos.Line,
		DotsColumn: dpos.Column,
		Body:       c.compile(reflect.ValueOf(stmt.Body)),
	}
}

// Match matches either a for statement or a range statement.
func (m ForDotsMatcher) Match(got reflect.Value, d data.Data) (data.Data, bool) {
	var (
		bodyField   forDotsField
		otherFields []forDotsField
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
		} else {
			otherFields = append(otherFields, f)
		}
	}

	d = data.WithValue(
		d,
		forDotsKey{Line: m.DotsLine, Column: m.DotsColumn},
		forDotsData{
			Type:         t,
			BodyFieldIdx: bodyField.Idx,
			OtherFields:  otherFields,
		},
	)

	return m.Body.Match(bodyField.Value, d)
}

// ForDotsReplacer replaces a "for ...".
type ForDotsReplacer struct {
	DotsLine, DotsColumn int // line and column on which dots appear
	Body                 Replacer
}

func (c *replacerCompiler) compileForStmt(v reflect.Value) Replacer {
	stmt := v.Interface().(*ast.ForStmt)

	if _, isDots := stmt.Cond.(*pgo.Dots); !isDots || stmt.Init != nil || stmt.Post != nil {
		// Not a "for ...". Fall back to the usual logic.
		return c.compileGeneric(v)
	}

	dpos := c.fset.Position(stmt.Cond.Pos())
	return ForDotsReplacer{
		DotsLine:   dpos.Line,
		DotsColumn: dpos.Column,
		Body:       c.compile(reflect.ValueOf(stmt.Body)),
	}
}

// Replace rebuilds a For or Range statement from the originally captured
// fields.
func (r ForDotsReplacer) Replace(d data.Data) (reflect.Value, error) {
	var fd forDotsData
	if !data.Lookup(d, forDotsKey{Line: r.DotsLine, Column: r.DotsColumn}, &fd) {
		return reflect.Value{}, errors.New("match data not found for 'for ...': " +
			"are you sure that the line appears on a context line without a preceding '-' or '+'?")
	}

	stmt := reflect.New(fd.Type).Elem()

	// Reproduce fields besides the body as-is.
	for _, f := range fd.OtherFields {
		stmt.Field(f.Idx).Set(f.Value)
	}

	body, err := r.Body.Replace(d)
	if err != nil {
		return reflect.Value{}, err
	}

	stmt.Field(fd.BodyFieldIdx).Set(body)
	return stmt.Addr(), nil
}

type forDotsKey struct{ Line, Column int }

type forDotsData struct {
	// Type of statement that we matched.
	//
	// This is one of ForStmt or RangeStmt.
	Type reflect.Type

	// Field index in Type at which the "Body *ast.BlockStmt" field is
	// present.
	BodyFieldIdx int

	OtherFields []forDotsField
}

type forDotsField struct {
	Idx   int           // index of the field in Type
	Value reflect.Value // captured value of the field
}
