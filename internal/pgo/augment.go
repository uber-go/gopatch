package pgo

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/uber-go/gopatch/internal/pgo/augment"
	"go.uber.org/multierr"
	"golang.org/x/tools/go/ast/astutil"
)

// augmentAST replaces portions of this AST with pgo nodes based on the
// provided augmentations.
func augmentAST(file *token.File, n ast.Node, augs []augment.Augmentation, adj *posAdjuster) (ast.Node, error) {
	a := newAugmenter(file, augs, adj)
	n = astutil.Apply(n, a.Apply, nil)
	return n, a.Err()
}

var (
	astExprType  = reflect.TypeOf((*ast.Expr)(nil)).Elem()
	astStmtType  = reflect.TypeOf((*ast.Stmt)(nil)).Elem()
	astFieldType = reflect.TypeOf((*ast.Field)(nil))
)

// Augmenter is used to traverse the Go AST and replace sections of it.
type augmenter struct {
	file   *token.File
	errors []error

	// Used to adjust positions back to the original file.
	adj *posAdjuster

	// augmentations are indexed by position so that we can match them to
	// nodes as we traverse.
	augs map[augPos]augment.Augmentation
}

// augPos is the position of an augmentation in the file.
type augPos struct {
	// Start and end offsets of the augmentation.
	start, end int
}

func newAugmenter(file *token.File, augs []augment.Augmentation, adj *posAdjuster) *augmenter {
	index := make(map[augPos]augment.Augmentation, len(augs))
	for _, aug := range augs {
		start, end := aug.Start(), aug.End()
		index[augPos{start: start, end: end}] = aug
	}
	return &augmenter{
		file: file,
		augs: index,
		adj:  adj,
	}
}

// errf logs errors at the given position. Position must be in the augmented
// file, not the original file.
func (a *augmenter) errf(pos token.Pos, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	a.errors = append(a.errors, fmt.Errorf("%v: %v", a.adj.Position(pos), msg))
}

// pop retrieves the augmentation associated with the provided node, or nil if
// no augmentation applies to this node. If an augmentation was found, it is
// removed from the list of known augmentations.
func (a *augmenter) pop(n ast.Node) augment.Augmentation {
	off := augPos{
		start: a.file.Offset(n.Pos()),
		end:   a.file.Offset(n.End()),
	}
	aug, ok := a.augs[off]
	if ok {
		// Augmentations are deleted so that we can track unused augmentations
		// later.
		delete(a.augs, off)
	}
	return aug
}

// Apply visits the given AST cursor. Use this with astutil.Apply.
func (a *augmenter) Apply(cursor *astutil.Cursor) bool {
	n := cursor.Node()
	if n == nil {
		return false
	}

	aug := a.pop(n)
	if aug == nil {
		return true // keep looking
	}

	// We can't rely on cursor.Replace because we want to inspect if the type
	// of the target field matches the kind of expression we are placing
	// there. We can't do this with type switches because for interfaces, type
	// switches will match if the target implements the interface.
	//
	// For example, for a ast.TypeSpec, the Name field is of type *ast.Ident.
	// To determine if we can place a &Dots{} there, we need ask the type
	// system if that field is an ast.Expr (which it's not). The type system
	// will respond with yes because *ast.Ident implements ast.Expr.
	//
	// Instead, we'll use reflection to inspect the type of the field and then
	// let cursor.Replace handle it.
	parent := reflect.TypeOf(cursor.Parent())
	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}
	field, _ := parent.FieldByName(cursor.Name())
	fieldType := field.Type
	if cursor.Index() >= 0 {
		fieldType = fieldType.Elem()
	}

	switch aug := aug.(type) {
	case *augment.Dots:
		dots := &Dots{Dots: n.Pos()}
		switch fieldType {
		case astStmtType:
			cursor.Replace(&ast.ExprStmt{X: dots})
		case astFieldType:
			cursor.Replace(&ast.Field{Type: dots})
		case astExprType:
			cursor.Replace(dots)
		default:
			a.errf(n.Pos(), `found unexpected "..." inside %T`, n)
			return false
		}
	default:
		panic(fmt.Sprintf("unknown augmentation %T", aug))
	}

	return false
}

// Err returns an error if this augmenter encountered any errors or if any of
// the augmentations were unused.
func (a *augmenter) Err() error {
	for _, aug := range a.augs {
		a.errf(a.file.Pos(aug.Start()), "unused augmentation %T", aug)
	}
	return multierr.Combine(a.errors...)
}
