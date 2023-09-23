package rand

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRandStringBytes(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Not empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RandStringBytes(tt.args.n)
			require.Len(t, got, tt.args.n)
		})
	}
}
