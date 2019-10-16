package main

import "errors"

var (
	ErrExpectedKeyword        = errors.New("expected keyword")
	ErrExpectedIdentifier     = errors.New("expected identifier")
	ErrExpectedOpeningBracket = errors.New("expected opening bracket")
	ErrExpectedClosingBracket = errors.New("expected closing bracket")
	ErrExpectedArguments      = errors.New("expected arguments")
	ErrExpectedArgument       = errors.New("expected argument")
	ErrExpectedType           = errors.New("expected type")
	ErrExpectedDelimiter      = errors.New("expected delimiter")
	ErrUnexpectedError        = errors.New("unexpected error")
)

type Statement struct {
	Keyword ExpressionType // function
	Name    string         // function
	Type    string         // argument

	Arguments []*Statement // function

	Body []*Statement // bracket
}

type parserFn func(*parser, *Statement) (*Statement, parserFn, error)

type parser struct {
	items chan Expression

	stmts       chan *Statement
	stmtsLength int

	backlog       chan Expression
	backlogLength int
}

func (p *parser) next() (exp Expression, ok bool) {
	if p.backlogLength > 0 {
		exp, ok = <-p.backlog
		p.backlogLength--
		return
	}

	exp, ok = <-p.items
	return
}

func (p *parser) push(s *Statement) {
	stmts := make(chan *Statement, p.stmtsLength+1)
	stmts <- s
	for i := 0; i < p.stmtsLength; i++ {
		stmts <- <-p.stmts
	}
	p.stmts = stmts
	p.stmtsLength++
}

func (p *parser) pop() (s *Statement) {
	if p.stmtsLength > 0 {
		s = <-p.stmts
		p.stmtsLength--
	}

	return
}

func (p *parser) backward(exp Expression) {
	backlog := make(chan Expression, p.backlogLength+1)
	backlog <- exp
	for i := 0; i < p.backlogLength; i++ {
		backlog <- <-p.backlog
	}
	p.backlog = backlog
	p.backlogLength++
}

func ParseASTTree(items chan Expression) (*Statement, error) {
	p := &parser{
		items: items,
	}

	s, _, err := parseBracket(p, &Statement{})
	return s, err
}

func parseBracket(p *parser, s *Statement) (*Statement, parserFn, error) {
	exp, ok := p.next()
	if !ok {
		return nil, nil, ErrUnexpectedError
	}
	if exp.Type() == EOF {
		return s, nil, nil
	}

	var fn parserFn
	switch exp.Type() {
	case Kfunc:
		fn = parseFunc
		break
	}

LOOP:
	_s := &Statement{}
	if s.Body == nil {
		s.Body = []*Statement{_s}
	} else {
		s.Body = append(s.Body, _s)
	}

	var err error
	var n *Statement
	for n, fn, err = fn(p, _s); fn != nil && err == nil; n, fn, err = fn(p, n) {
	}
	if err != nil {
		return nil, nil, err
	}

	exp, ok = p.next()
	if ok {
		p.backward(exp)
		fn = parseBracket
		goto LOOP
	}

	return s, nil, nil
}

func parseFunc(p *parser, _s *Statement) (*Statement, parserFn, error) {

	// Keyword: Kfunc -> function definition
	exp, ok := p.next()
	if !ok || exp.Type() != Kfunc {
		return nil, nil, ErrExpectedKeyword
	}
	s := &Statement{
		Keyword: exp.Type(),
	}

	// Keyword: Tidentifier -> function name
	exp, ok = p.next()
	if !ok || exp.Type() != Tidentifier {
		return nil, nil, ErrExpectedKeyword
	}
	s.Name = exp.Expression()

	// Keyword: Tstart -> function args open
	exp, ok = p.next()
	if !ok || exp.Type() != Tstart {
		return nil, nil, ErrExpectedOpeningBracket
	}

	// Keyword: Tclose -> function args close
	// Loop over args beforehand
	args := []*Statement{}
	for exp, ok = p.next(); ok || exp.Type() != Tclose; exp, ok = p.next() {

		// Keyword: Tidentifier -> name of argument
		if exp.Type() != Tidentifier {
			return nil, nil, ErrExpectedArgument
		}
		a := &Statement{
			Name: exp.Expression(),
		}

		// Keyword: Ktype -> type of argument
		exp, ok = p.next()
		if !ok || exp.Type() != Ktype {
			return nil, nil, ErrExpectedType
		}
		a.Type = exp.Expression()
		args = append(args, a)

		// Keyword: Tliteral -> , delimiter
		// Keyword: Tclose -> ) delimiter
		exp, ok = p.next()
		if !ok || exp.Type() != Tliteral || exp.Expression() != "," {
			if exp.Type() == Tclose {
				p.backward(exp)
				continue
			}

			return nil, nil, ErrExpectedDelimiter
		}
	}
	if !ok {
		return nil, nil, ErrExpectedArguments
	}
	s.Arguments = args

	s, _, err := parseBracket(p, s)
	if err != nil {
		return nil, nil, err
	}
	exp, ok = p.next()
	if !ok || exp.Type() != Tclose {
		return nil, nil, ErrExpectedClosingBracket
	}

	return s, nil, err
}
