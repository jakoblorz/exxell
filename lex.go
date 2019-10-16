package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Expression interface {
	Type() ExpressionType
	Expression() string
	String() string
}

type item struct {
	key ExpressionType
	val string
}

func (i item) Type() ExpressionType {
	return i.key
}

func (i item) Expression() string {
	return i.val
}

func (i item) String() string {
	return fmt.Sprintf("%d \"%s\"", i.key, i.val)
}

type lexer struct {
	input string

	pos    int
	start  int
	width  int
	adepth int
	bdepth int

	caller lexerFn
	items  chan Expression
}

type lexerFn func(*lexer) lexerFn

func LexExpression(input string) []Expression {
	expressionCh := make(chan Expression)
	go ParseExpressions(input, expressionCh)

	expressions := []Expression{}
	for {
		exp, ok := <-expressionCh
		if !ok {
			break
		}

		expressions = append(expressions, exp)
	}

	return expressions
}

func ParseExpressions(input string, items chan Expression) {
	l := &lexer{
		input: input,
		items: items,
	}
	l.run()
}

func (l *lexer) run() {
	for state := lexBracket; state != nil; {
		l.caller = state
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t ExpressionType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) forward() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) backward() {
	l.pos -= l.width
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) peek() rune {
	r := l.forward()
	l.backward()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.forward()) != -1 {
		return true
	}
	l.backward()
	return false
}

func (l *lexer) skip(valid string) {
	for strings.IndexRune(valid, l.forward()) != -1 {
	}
	l.backward()
}

func (l *lexer) errorf(format string, args ...interface{}) lexerFn {
	l.items <- item{
		Terror,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) atTerminator() bool {
	r := l.peek()
	if isSpace(r) || isEndOfLine(r) {
		return true
	}
	switch r {
	case eof, '.', ',', '|', ':', ')', '(':
		return true
	}
	// Does r start the delimiter? This can be ambiguous (with delim=="//", $x/2 will
	// succeed but should fail) but only in extremely rare cases caused by willfully
	// bad choice of delimiter.
	// if rd, _ := utf8.DecodeRuneInString(l.rightDelim); rd == r {
	// 	return true
	// }
	return false
}
func lexDelimiter(l *lexer, t ExpressionType, delim string, then lexerFn, includeDelim bool) lexerFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], delim) {
			if includeDelim {
				l.pos += len(delim)
			}
			if l.pos > l.start {
				l.emit(t)
			}
			if !includeDelim {
				l.pos += len(delim)
				l.ignore()
			}
			return then
		}
		if l.forward() == eof {
			break
		}
	}
	if l.pos > l.start {
		l.emit(t)
	}
	l.emit(EOF)
	return nil
}

func lexLineComment(l *lexer) lexerFn {
	return lexDelimiter(l, Tcomment, string('\n'), lexBracket, false)
}

func lexBlockComment(l *lexer) lexerFn {
	return lexDelimiter(l, Tcomment, `*/`, lexBracket, true)
}

func lexNumber(l *lexer) lexerFn {
	l.accept("+-")
	digits := "0123456789"
	if l.accept("0") {
		if l.accept("xX") {
			digits = "123456789abcdefABCDEF"
		} else if l.accept("oO") {
			digits = "01234567_"
		} else if l.accept("bB") {
			digits = "01_"
		}
	}
	l.skip(digits)
	if l.accept(".") {
		l.skip(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.skip("0123456789")
	}
	l.accept("i")
	if isAlphaNumeric(l.peek()) {
		l.forward()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	l.emit(Tnumber)
	return lexBracket
}

func lexQuote(l *lexer) lexerFn {
	return lexDelimiter(l, Tquote, "`", lexBracket, false)
}

func lexString(l *lexer) lexerFn {
	return lexDelimiter(l, Tstring, `"`, lexBracket, false)
}

func lexSpace(l *lexer) lexerFn {
	var r rune
	var numSpaces int
	for {
		r = l.peek()
		if !isSpace(r) {
			break
		}
		l.forward()
		numSpaces++
	}

	// toggle later?
	if true {
		l.ignore()
	} else {
		l.emit(Tspace)
	}
	return lexBracket
}

func lexEndOfLine(l *lexer) lexerFn {
	var r rune
	var numLines int
	for {
		r = l.peek()
		if !isEndOfLine(r) {
			break
		}
		l.forward()
		numLines++
	}

	// toggle later?
	if true {
		l.ignore()
	} else {
		l.emit(EOL)
	}
	return lexBracket
}

func lexChar(l *lexer) lexerFn {
Loop:
	for {
		switch l.forward() {
		case '\\':
			if r := l.forward(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated character constant")
		case '\'':
			break Loop
		}
	}
	l.emit(Tchar)
	return lexBracket
}

func lexIdentifier(l *lexer) lexerFn {
Loop:
	for {
		switch r := l.forward(); {
		case isAlphaNumeric(r):
		case r == '.':

		default:
			l.backward()
			word := l.input[l.start:l.pos]
			if !l.atTerminator() {
				return l.errorf("unexpected character %#U", r)
			}
			switch {
			case key[word] > KeywordStop:
				l.emit(key[word])
			case word == "true", word == "false":
				l.emit(Tbool)
			default:
				l.emit(Tidentifier)
			}
			break Loop
		}
	}
	return lexBracket
}

func lexBracket(l *lexer) (next lexerFn) {
	switch r := l.forward(); {

	case r == eof:
		if l.adepth != 0 || l.bdepth != 0 {
			return l.errorf("unclosed bracket")
		}
		return nil

	case isEndOfLine(r):
		l.backward()
		return lexEndOfLine
	case isSpace(r):
		l.backward()
		return lexSpace

	case r == '"':
		l.ignore()
		return lexString
	case r == '`':
		l.ignore()
		return lexQuote
	case r == '\'':
		return lexChar

	case r == '/':
		switch r2 := l.forward(); {
		case r2 == '/':
			return lexLineComment
		case r2 == '*':
			return lexBlockComment
		default:
			l.backward()
		}
		fallthrough

	case r == '+' || r == '-' || ('0' <= r && r <= '9'):
		l.backward()
		return lexNumber
	case isAlphaNumeric(r):
		l.backward()
		return lexIdentifier

	case r == '(':
		l.emit(Tstart)
		l.adepth++
	case r == ')':
		l.emit(Tclose)
		l.adepth--
		if l.adepth < 0 {
			return l.errorf("unexpected character %#U", r)
		}

	case r == '{':
		l.emit(Tstart)
		l.bdepth++
	case r == '}':
		l.emit(Tclose)
		l.bdepth--
		if l.bdepth < 0 {
			return l.errorf("unexpected charachter %#U", r)
		}

	case r < unicode.MaxASCII && unicode.IsPrint(r):
		l.emit(Tliteral)
		return lexBracket

	default:
		return l.errorf("unrecognized character %#U", r)
	}
	return lexBracket
}

// isSpace reports whether r is a Tspace TCharacter.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an Taclose-of-line TCharacter.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
