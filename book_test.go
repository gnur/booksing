package booksing

import "testing"

func Test_fix(t *testing.T) {
	type args struct {
		s            string
		capitalize   bool
		correctOrder bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "author with comma",
			args: args{
				s:            "brown, dan  ",
				capitalize:   true,
				correctOrder: true,
			},
			want: "Dan Brown",
		},
		{
			name: "author without comma but lots of whitespace",
			args: args{
				s:            "  dan  brown  ",
				capitalize:   true,
				correctOrder: true,
			},
			want: "Dan Brown",
		},
		{
			name: "title with year",
			args: args{
				s:            "1984 (2001)",
				capitalize:   true,
				correctOrder: false,
			},
			want: "1984",
		},
		{
			name: "title with druk",
			args: args{
				s:            "1984 / druk 2",
				capitalize:   true,
				correctOrder: false,
			},
			want: "1984",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Fix(tt.args.s, tt.args.capitalize, tt.args.correctOrder); got != tt.want {
				t.Errorf("fix() = %v, want %v", got, tt.want)
			}
		})
	}
}
