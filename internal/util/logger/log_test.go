package logger

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name: `Test debug`,
			args: args{level: DEBUG, message: `Some message`},
			want: want{message: struct {
				part1 string
				part2 string
			}{part1: `DEBUG`, part2: `Some message`}},
		},
		{
			name: `Test warning`,
			args: args{level: WARN, message: `Some message`},
			want: want{message: struct {
				part1 string
				part2 string
			}{part1: `WARN`, part2: `Some message`}},
		},
		{
			name: `Test fatal`,
			args: args{level: FATAL, message: `Some message`},
			want: want{message: struct {
				part1 string
				part2 string
			}{part1: `FATAL`, part2: `Some message`}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err)

			rescueStdout := os.Stdout
			os.Stdout = w

			PrintLog(tt.args.level, tt.args.message, true)

			err = w.Close()
			require.NoError(t, err)
			out, _ := io.ReadAll(r)
			os.Stdout = rescueStdout

			assert.Contains(t, string(out), tt.want.message.part1)
			assert.Contains(t, string(out), tt.want.message.part2)
			require.NoError(t, err)
		})
	}
}
