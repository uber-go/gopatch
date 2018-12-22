package parse

import (
	"fmt"
	"go/ast"
	"go/scanner"
	"go/token"

	"github.com/uber-go/gopatch/internal/parse/section"
	"go.uber.org/multierr"
)

// Parses the metavariables section of the change at index i.
func (p *parser) parseMeta(i int, c *section.Change) (*Meta, error) {
	metaContents, metaLines := section.ToBytes(c.Meta)

	// We will create a new File with the contents of the metavariables
	// section and map positions in it back to the original file for error
	// messages.

	// Generate a fake name for the File.
	filename := p.fset.File(c.Pos()).Name()
	if len(c.Name) > 0 {
		filename += c.Name + ".meta"
	} else {
		filename += fmt.Sprintf("%d.meta", i)
	}

	file := p.fset.AddFile(filename, -1, len(metaContents))
	for _, line := range metaLines {
		p := p.fset.Position(line.Pos)
		file.AddLineColumnInfo(line.Offset, p.Filename, p.Line, p.Column)
	}

	parser := metaParser{fset: p.fset}
	var scanner scanner.Scanner
	scanner.Init(file, metaContents, parser.onError, 0 /* mode */)
	parser.scanner = &scanner
	parser.next() // read the first token

	return parser.parse(), multierr.Combine(parser.errors...)
}

type metaParser struct {
	scanner *scanner.Scanner

	fset *token.FileSet
	pos  token.Pos   // current token position
	tok  token.Token // current token
	text string      // current token contents

	failed bool
	errors []error
}

// This function is called by go/scanner when errors are encountered. We
// connect it in the Init call above.
func (p *metaParser) onError(pos token.Position, msg string) {
	p.failed = true
	p.errors = append(p.errors, fmt.Errorf("%v: %v", pos, msg))
}

// Posts a formatted error message to the parser.
func (p *metaParser) errf(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	p.onError(p.fset.Position(p.pos), msg)
}

// Advances to the next token.
func (p *metaParser) next() {
	p.pos, p.tok, p.text = p.scanner.Scan()
}

// Parses the metavariables section.
func (p *metaParser) parse() *Meta {
	var m Meta
	for !p.failed && p.tok != token.EOF {
		m.Vars = append(m.Vars, p.parseDecl())
	}
	return &m
}

// Parses and returns a VarDecl.
//
//   var x, y, z Foo
func (p *metaParser) parseDecl() *VarDecl {
	defer p.next()

	if p.tok != token.VAR {
		p.errf(`unexpected %q, expected "var"`, p.tok)
		return nil
	}

	d := VarDecl{VarPos: p.pos}
	for {
		p.next() // skip var/,
		name := p.parseIdent()
		if name == nil {
			return nil
		}
		d.Names = append(d.Names, name)

		if p.tok != token.COMMA {
			break
		}
	}

	// A type name is expected after list of variables.
	d.Type = p.parseIdent()
	if d.Type == nil {
		return nil
	}

	// go/scanner implicitly inserts SEMICOLON when a newline is found where a
	// semicolon would be accepted. So we expect a semicolon after every var
	// declaration.
	if p.tok != token.SEMICOLON {
		p.errf(`unexpected %q, expected ";" or a newline`, p.tok)
		return nil
	}

	return &d
}

// Reads and returns an identifier, advancing the parser to the next token.
// Fails the parser and returns nil if an identifier was not found.
func (p *metaParser) parseIdent() *ast.Ident {
	defer p.next()

	if p.tok != token.IDENT {
		p.errf("unexpected %q, expected an identifier", p.tok)
		return nil
	}

	return &ast.Ident{Name: p.text, NamePos: p.pos}
}
