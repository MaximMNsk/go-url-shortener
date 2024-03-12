package logger

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestPrintLog(t *testing.T) {
	type args struct {
		level   string
		message string
	}
	type want struct {
		message struct {
			part1 string
			part2 string
		}
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test error`,
			args: args{level: ERROR, message: `Some message`},
			want: want{message: struct {
				part1 string
				part2 string
			}{part1: `ERROR`, part2: `Some message`}},
		},
		{
			name: `Test info`,
			args: args{level: INFO, message: `Some message`},
			want: want{message: struct {
				part1 string
				part2 string
			}{part1: `INFO`, part2: `Some message`}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, err := os.Pipe()
			rescueStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintLog(tt.args.level, tt.args.message)

			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = rescueStdout

			assert.Contains(t, string(out), tt.want.message.part1)
			assert.Contains(t, string(out), tt.want.message.part2)
			require.NoError(t, err)
		})
	}

}
