package shorter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_GetShortURL(t *testing.T) {
	type args struct {
		linkID   string
		hostPort string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test link",
			args: args{linkID: "0X0X0X", hostPort: "http://localhost:8080"},
			want: "http://localhost:8080/0X0X0X",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetShortURL(tt.args.hostPort, tt.args.linkID), "GetShortURL(%v)", tt.args.linkID)
		})
	}
}
