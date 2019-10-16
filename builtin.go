package main

type ExpressionType int

const (
	Terror ExpressionType = iota
	Tliteral
	Tbool
	Tchar
	EOF
	EOL
	Tstart
	Tclose
	Tnumber
	Tspace
	Tidentifier
	Tquote
	Tstring
	Tcomment

	KeywordStop
	Ktype
	Kfunc
	Klet
	Kset
	Kif

	eof rune = -1
)

var key = map[string]ExpressionType{

	// Assignation
	":=": Klet,
	"=":  Kset,

	// True Keywords
	"if":   Kif,
	"func": Kfunc,

	// Types
	"byte":     Ktype,
	"int16":    Ktype,
	"int32":    Ktype,
	"int64":    Ktype,
	"decimal":  Ktype,
	"currency": Ktype,
	"date":     Ktype,
	"string":   Ktype,
	"charset":  Ktype,
	"object":   Ktype,
	"bool":     Ktype,
}
