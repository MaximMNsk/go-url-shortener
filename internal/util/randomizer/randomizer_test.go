package randomizer

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestRandDigitalBytes(t *testing.T) {
	type args struct {
		length int
	}
	type want struct {
		min int
		max int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test length`,
			args: args{length: 6},
			want: want{min: 1, max: 999999},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxI := math.Pow(10, float64(tt.args.length))
			for i := 0; i <= int(maxI); i++ {
				x, err := RandDigitalBytes(tt.args.length)
				assert.LessOrEqual(t, tt.want.min, x)
				assert.LessOrEqual(t, x, tt.want.max)
				require.NoError(t, err)
			}
		})
	}

}
