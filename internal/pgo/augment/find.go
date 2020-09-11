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
		f.process()
	}

	return f.augs
}

func (f *finder) process() {
	switch f.tok {
	case token.IDENT:
		f.ident()
	case token.ELLIPSIS:
		f.ellipsis()
	case token.LSS:
		f.lss()
	case token.FUNC:
		f.function()
	default:
		f.next()
	}
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
	case token.TYPE, token.CONST, token.VAR:
		// We can parse these as GenDecls. These transformations will
		// apply to both, top-level GenDecls and GenDecls nested
		// inside DeclStmts.
		//
		// If users want to place multiple of these in the patch, they
		// should use {} to ensure that the patch is interpreted as a
		// list of statemetns.
		f.next() // type/const/var
	case token.FUNC:
		f.funcDecl()
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

	// ...>
	if f.tok == token.GTR && f.pos == pos+3 && sameLine {
		f.append(&RDots{
			RDotsStart: off,
			RDotsEnd:   off + 4,
		})
		f.next() // >
		return
	}

	// ...
	f.append(&Dots{DotsStart: off, DotsEnd: off + 3})
}

// Processes a top-level function or method declaration.
func (f *finder) funcDecl() {
	f.next() // func

	// handle receiver if present
	if f.tok == token.LPAREN {
		f.next() // (

		for f.tok != token.RPAREN {
			f.process()
		}
		f.next() // )
	}

	f.next() // func name

	f.params()
	f.results()
}

// Processes a function literal.
func (f *finder) function() {
	f.next() // func
	f.params()
	f.results()
}

func (f *finder) params() {
	f.fieldList()
}

func (f *finder) results() {
	if f.tok == token.LPAREN {
		f.fieldList()
	}
	// return to process loop for unwrapped results
}

// Processes a argument or results lists.
func (f *finder) fieldList() {
	f.next() // (
	var (
		ellipses []int // list of offsets at which ellipses were found
		named    bool
	)

	for f.tok != token.RPAREN {
		switch f.tok {
		case token.FUNC:
			f.function()
		case token.IDENT:
			f.next() // ident
			if f.tok == token.PERIOD {
				// ident was beginning of selector expression; pop off
				// the "." and next ident before checking if named.
				f.next() // .
				f.next() // ident
			}

			// For the next token,
			//
			// - comma indicates that a new parameter is beginning
			// - rparen indicates the end of the parameter list
			//
			// If either token appears after a single identifier, this was an unnamed
			// parameter. So anything else as the next token indicates a named
			// parameter.
			if f.tok != token.COMMA && f.tok != token.RPAREN {
				named = true
			}
		case token.ELLIPSIS:
			off := f.offset
			f.next() // ...
			if f.tok == token.IDENT {
				// ellipsis was variadic operator, continue without
				// augmenting this ellipsis.
				continue
			}

			ellipses = append(ellipses, off)
		default:
			// * or , e.g.
			f.next()
		}
	}
	f.next() // )

	for _, off := range ellipses {
		f.append(&Dots{DotsStart: off, DotsEnd: off + 3, Named: named})
	}
}

func (f *finder) lss() {
	pos := f.pos
	off := f.offset
	f.next() // <

	sameLine := f.line(pos) == f.line(f.pos)

	// <...
	if f.tok == token.ELLIPSIS && f.pos == pos+1 && sameLine {
		f.append(&LDots{
			LDotsStart: off,
			LDotsEnd:   off + 4,
		})
		f.next() // ...
		return
	}
}
