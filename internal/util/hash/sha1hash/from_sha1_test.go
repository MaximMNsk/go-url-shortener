package sha1hash

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreate(t *testing.T) {
	type args struct {
		input string
		len   int
	}
	type want struct {
		output string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Hash aaa`,
			args: args{
				input: `aaa`,
				len:   10,
			},
			want: want{
				output: `7e240de74f`,
			},
		},
		{
			name: `Test Hash nil`,
			args: args{
				input: ``,
				len:   10,
			},
			want: want{
				output: `da39a3ee5e`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Create(tt.args.input, tt.args.len)
			assert.Equal(t, tt.want.output, a)
		})
	}
}
