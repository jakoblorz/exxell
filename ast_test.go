package main

import (
	"reflect"
	"testing"
)

func TestParseASTTree(t *testing.T) {
	items := make(chan Expression)
	go ParseExpressions(testProgram, items)

	type args struct {
		items chan Expression
	}
	tests := []struct {
		name    string
		args    args
		want    *Statement
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				items: items,
			},
			want:    &Statement{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ /* err */ := ParseASTTree(tt.args.items)
			// if (err != nil) != tt.wantErr {
			// 	t.Errorf("ParseASTTree() error = %v, wantErr %v", err, tt.wantErr)
			// 	return
			// }
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseASTTree() = %v, want %v", got, tt.want)
			}
		})
	}
}
