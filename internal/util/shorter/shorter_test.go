package shorter

import (
	random "github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/server/config"
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

func BenchmarkGetShortURL(b *testing.B) {
	var Cfg config.OuterConfig
	_ = Cfg.InitConfig(true)
	count := 10000
	type args struct {
		addr   string
		linkID string
	}
	b.Run(`GetShortURL`, func(b *testing.B) {
		args := args{
			addr:   Cfg.Final.AppAddr,
			linkID: random.StringBytes(10),
		}
		for i := 0; i < count; i++ {
			_ = GetShortURL(args.addr, args.linkID)
		}
	})
}
