package memory

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var Store MemStorage

func TestMemStorage_Init(t *testing.T) {
	type want struct {
		MemErr ErrorMemory
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				MemErr: ErrorMemory{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `prepare`,
					message:        ``,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config.OuterConfig
			ConfErr := cfg.InitConfig(true)
			require.NoError(t, ConfErr)
			Store = MemStorage{
				Cfg: cfg,
			}

			err := Store.Init()
			require.NoError(t, err)
		})
	}
}

func TestMemStorage_Ping(t *testing.T) {
	type want struct {
		pingRes bool
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				pingRes: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config.OuterConfig
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			res, err := Store.Ping(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want.pingRes, res)
		})
	}
}

func TestMemStorage_Set(t *testing.T) {
	var cfg config.OuterConfig
	errCfg := cfg.InitConfig(true)

	type args struct {
		link      string
		shortLink string
		hashLink  string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Set",
			args: args{
				link:      `aaa`,
				shortLink: `http://localhost:8080:bbb`,
				hashLink:  `bbb`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, errCfg)

			err := Store.Set(context.Background(), tt.args.link, tt.args.shortLink, tt.args.hashLink, 0)
			require.NoError(t, err)
		})
	}
}

func TestMemStorage_Get(t *testing.T) {
	var cfg config.OuterConfig
	errCfg := cfg.InitConfig(true)

	type want struct {
		link      string
		shortLink string
	}
	type args struct {
		fileName string
		hashLink string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Get",
			args: args{
				hashLink: `bbb`,
			},
			want: want{
				link:      `aaa`,
				shortLink: `http://localhost:8080:bbb`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, errCfg)

			link, _, err := Store.Get(context.Background(), tt.args.hashLink)
			require.NoError(t, err)
			require.Equal(t, tt.want.link, link)
		})
	}
}
