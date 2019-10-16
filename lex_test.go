package main

import (
	"reflect"
	"testing"
)

var testProgram = `

func Main() {
	MySub("hi there")
}

func MySub(input string, x int16) {

}
`

func TestLexExpression(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want []Expression
	}{
		{
			name: "test",
			args: args{
				input: testProgram,
			},
			want: make([]Expression, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LexExpression(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LexExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
