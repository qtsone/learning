package gql

import (
	"fmt"
	"strconv"
	"strings"
)

// This file parses a deliberately small subset of the GraphQL query language:
// one operation, named fields, literal arguments, nested selection sets.
//
// Not supported, on purpose: variables, fragments, aliases, directives,
// subscriptions, multiple operations per document. Each of those is a real
// feature with real consequences — fragments in particular make depth analysis
// harder, because you cannot know a query's depth without expanding them —
// and the lesson names them where they matter. You are here to learn the
// execution model, not to write a parser.

type OperationType string

const (
	OperationQuery    OperationType = "query"
	OperationMutation OperationType = "mutation"
)

// Operation is a parsed request: one query or one mutation.
type Operation struct {
	Type       OperationType
	Name       string
	Selections []*Field
}

// Field is one entry in a selection set, with its arguments and whatever it
// selects underneath.
type Field struct {
	Name       string
	Args       Args
	Selections []*Field
}

// Parse turns a query string into an Operation. Every failure here is a
// *request* error: nothing executed, so the response carries errors and no
// data at all.
func Parse(src string) (*Operation, error) {
	p := &parser{lex: &lexer{src: src}}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return p.parseOperation()
}

// ---------------------------------------------------------------- lexer

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokPunct
	tokName
	tokInt
	tokString
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) skipIgnored() {
	for l.pos < len(l.src) {
		switch c := l.src[l.pos]; {
		case c == ' ', c == '\t', c == '\n', c == '\r', c == ',':
			l.pos++
		case c == '#':
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
		default:
			return
		}
	}
}

func isDigit(c byte) bool     { return c >= '0' && c <= '9' }
func isNameStart(c byte) bool { return c == '_' || (c|0x20) >= 'a' && (c|0x20) <= 'z' }
func isNameCont(c byte) bool  { return isNameStart(c) || isDigit(c) }

func (l *lexer) next() (token, error) {
	l.skipIgnored()
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	switch {
	case strings.IndexByte("{}():", c) >= 0:
		l.pos++
		return token{kind: tokPunct, text: string(c), pos: start}, nil

	case isNameStart(c):
		for l.pos < len(l.src) && isNameCont(l.src[l.pos]) {
			l.pos++
		}
		return token{kind: tokName, text: l.src[start:l.pos], pos: start}, nil

	case c == '-' || isDigit(c):
		l.pos++
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.pos++
		}
		return token{kind: tokInt, text: l.src[start:l.pos], pos: start}, nil

	case c == '"':
		l.pos++
		for l.pos < len(l.src) && l.src[l.pos] != '"' {
			if l.src[l.pos] == '\\' {
				l.pos++
			}
			l.pos++
		}
		if l.pos >= len(l.src) {
			return token{}, fmt.Errorf("unterminated string at offset %d", start)
		}
		l.pos++
		text, err := strconv.Unquote(l.src[start:l.pos])
		if err != nil {
			return token{}, fmt.Errorf("bad string literal at offset %d", start)
		}
		return token{kind: tokString, text: text, pos: start}, nil

	case c == '$':
		return token{}, fmt.Errorf("variables are not supported by this executor (offset %d)", start)

	case c == '.':
		return token{}, fmt.Errorf("fragments are not supported by this executor (offset %d)", start)
	}
	return token{}, fmt.Errorf("unexpected character %q at offset %d", string(c), start)
}

// ---------------------------------------------------------------- parser

type parser struct {
	lex *lexer
	tok token
}

func (p *parser) advance() error {
	t, err := p.lex.next()
	if err != nil {
		return err
	}
	p.tok = t
	return nil
}

func (p *parser) isPunct(s string) bool {
	return p.tok.kind == tokPunct && p.tok.text == s
}

func (p *parser) expectPunct(s string) error {
	if !p.isPunct(s) {
		return fmt.Errorf("expected %q at offset %d", s, p.tok.pos)
	}
	return p.advance()
}

func (p *parser) parseOperation() (*Operation, error) {
	op := &Operation{Type: OperationQuery}
	if p.tok.kind == tokName {
		switch p.tok.text {
		case "query", "mutation":
			op.Type = OperationType(p.tok.text)
		case "subscription":
			return nil, fmt.Errorf("subscriptions are not served over this endpoint")
		default:
			return nil, fmt.Errorf("expected \"query\" or \"mutation\", found %q", p.tok.text)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.tok.kind == tokName {
			op.Name = p.tok.text
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}
	sels, err := p.parseSelectionSet()
	if err != nil {
		return nil, err
	}
	op.Selections = sels
	if p.tok.kind != tokEOF {
		return nil, fmt.Errorf("the request must contain exactly one operation (offset %d)", p.tok.pos)
	}
	return op, nil
}

func (p *parser) parseSelectionSet() ([]*Field, error) {
	if err := p.expectPunct("{"); err != nil {
		return nil, err
	}
	var out []*Field
	for !p.isPunct("}") {
		if p.tok.kind != tokName {
			return nil, fmt.Errorf("expected a field name at offset %d", p.tok.pos)
		}
		f, err := p.parseField()
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := p.advance(); err != nil { // past "}"
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty selection set at offset %d", p.tok.pos)
	}
	return out, nil
}

func (p *parser) parseField() (*Field, error) {
	f := &Field{Name: p.tok.text}
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.isPunct("(") {
		args, err := p.parseArgs()
		if err != nil {
			return nil, err
		}
		f.Args = args
	}
	if p.isPunct("{") {
		sels, err := p.parseSelectionSet()
		if err != nil {
			return nil, err
		}
		f.Selections = sels
	}
	return f, nil
}

func (p *parser) parseArgs() (Args, error) {
	if err := p.advance(); err != nil { // past "("
		return nil, err
	}
	args := Args{}
	for !p.isPunct(")") {
		if p.tok.kind != tokName {
			return nil, fmt.Errorf("expected an argument name at offset %d", p.tok.pos)
		}
		name := p.tok.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		if err := p.expectPunct(":"); err != nil {
			return nil, err
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		args[name] = v
	}
	return args, p.advance() // past ")"
}

func (p *parser) parseValue() (any, error) {
	t := p.tok
	switch {
	case t.kind == tokInt:
		n, err := strconv.Atoi(t.text)
		if err != nil {
			return nil, fmt.Errorf("bad integer %q at offset %d", t.text, t.pos)
		}
		return n, p.advance()
	case t.kind == tokString:
		return t.text, p.advance()
	case t.kind == tokName && (t.text == "true" || t.text == "false"):
		return t.text == "true", p.advance()
	case t.kind == tokName:
		return t.text, p.advance() // enum value, kept as its name
	}
	return nil, fmt.Errorf("expected a literal value at offset %d", t.pos)
}
