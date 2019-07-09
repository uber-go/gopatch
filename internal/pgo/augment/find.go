package augment

import (
	"fmt"
	"go/scanner"
	"go/token"

	"go.uber.org/multierr"
)

// find looks for augmentations to the Go source inside the given patch
// source.
func find(src []byte) ([]Augmentation, error) {
	fset := token.NewFileSet()
	file := fset.AddFile("src.go", -1, len(src))

	f := finder{file: file}
	var s scanner.Scanner
	s.Init(file, src, f.onError, 0 /* flags */)
	f.scanner = &s

	f.next() // read first token
	return f.find(), multierr.Combine(f.errors...)
}

type finder struct {
	file    *token.File
	scanner *scanner.Scanner

	tok token.Token // current token
	pos token.Pos   // position of current token

	// Offset of tok inside the original source file. This is equal to
	// file.Offset(pos).
	offset int

	// Augmentations and errors recorded so far.
	augs   []Augmentation
	errors []error
}

// Called by go/scanner in case of errors.
func (f *finder) onError(pos token.Position, msg string) {
	f.errors = append(f.errors, fmt.Errorf("%v: %v", pos, msg))
}

func (f *finder) append(aug Augmentation) {
	f.augs = append(f.augs, aug)
}

// Advances the scanner.
func (f *finder) next() {
	f.pos, f.tok, _ = f.scanner.Scan()
	f.offset = f.file.Offset(f.pos)
}

// Returns the line number for the provided token.Pos.
func (f *finder) line(pos token.Pos) int {
	return f.file.Line(pos)
}

func (f *finder) find() []Augmentation {
	f.pkg()
	f.imports()
	f.topLevelDecl()

	for f.tok != token.EOF {
		switch f.tok {
		case token.IDENT:
			f.ident()
		case token.ELLIPSIS:
			f.ellipsis()
		default:
			f.next()
		}
	}

	return f.augs
}

// Ensures that we have a package clause, recording the need for one if not.
func (f *finder) pkg() {
	if f.tok != token.PACKAGE {
		// Missing a package clause. Generate a fake one.
		f.append(&FakePackage{PackageStart: f.offset})
		return
	}

	f.next() // package
	f.next() // package_name
	f.next() // ;
}

// Skips over the imports, if any.
func (f *finder) imports() {
	for f.tok == token.IMPORT {
		f.next() // import

		switch f.tok {
		case token.LPAREN:
			// import group (import (...))
			// Skip until )
			for f.tok != token.RPAREN && f.tok != token.EOF {
				f.next()
			}
			f.next() // )
			f.next() // ;
			continue

		case token.PERIOD:
			// dot import (import . "foo")
			f.next() // .

		case token.IDENT:
			// named import (import x "foo")
			f.next() // x

		case token.EOF:
			// invalid syntax; parser will handle this
			return
		}

		f.next() // "foo"
		f.next() // ;
	}
}

// Ensures that we have a valid top-level decl.
func (f *finder) topLevelDecl() {
	switch f.tok {
	case token.FUNC, token.TYPE, token.CONST, token.VAR:
		// We can parse these as GenDecls. These transformations will
		// apply to both, top-level GenDecls and GenDecls nested
		// inside DeclStmts.
		//
		// If users want to place multiple of these in the patch, they
		// should use {} to ensure that the patch is interpreted as a
		// list of statemetns.
		f.next() // func/type/const
	case token.LBRACE:
		// Add a fake func()
		f.append(&FakeFunc{FuncStart: f.offset})
		f.next() // {
	default:
		// Add a fake func() { ... }
		f.append(&FakeFunc{FuncStart: f.offset, Braces: true})
	}
}

func (f *finder) ident() {
	f.next() // IDENT

	// foo...
	if f.tok == token.ELLIPSIS {
		f.next() // leave unchanged
	}
}

func (f *finder) ellipsis() {
	pos := f.pos
	off := f.offset
	f.next() // ...

	// The scanner tracks some parsing-related state, implicitly inserting
	// SEMICOLON tokens when newlines are encountered at the end of a
	// statement.
	//
	// This is problematic because ELLIPSIS is not a valid end of a statement.
	// So the following pgo code,
	//
	//   ...
	//   foo
	//
	// Will be scanned as, [ELLIPSIS, IDENT] rather than [ELLIPSIS, SEMICOLON,
	// IDENT], which makes it impossible to differentiate between that and
	// just "...foo" based on just tokens alone.
	//
	// To work around this, we need to check if the ELLIPSIS and IDENT are on
	// the same line.
	sameLine := f.line(pos) == f.line(f.pos)

	// ...foo
	if f.tok == token.IDENT && sameLine {
		f.next() // leave unchanged
		return
	}

	// ...
	f.append(&Dots{DotsStart: off, DotsEnd: off + 3})
}
