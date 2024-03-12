package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestHandleConfig(t *testing.T) {
	type args struct {
		envDb string
	}

	type wants struct {
		outputDbEnv     string
		outputDbDefault string
	}

	tests := []struct {
		name string
		args args
		want wants
	}{
		{
			name: `Test env`,
			args: args{envDb: `postgres://localhost`},
			want: wants{outputDbDefault: `postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable`, outputDbEnv: `postgres://localhost`},
		},
		{
			name: `Test default`,
			args: args{},
			want: wants{outputDbDefault: `postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == `Test env` {
				err := os.Setenv(`DATABASE_DSN`, tt.args.envDb)
				require.NoError(t, err)
				var cfg OuterConfig
				err = cfg.InitConfig(false)
				require.NoError(t, err)
				assert.Equal(t, tt.want.outputDbEnv, cfg.Final.DB)
			}
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != `Test default` {
				var cfg OuterConfig
				err := cfg.InitConfig(true)
				require.NoError(t, err)
				assert.Equal(t, tt.want.outputDbDefault, cfg.Final.DB)
			}
		})
	}
}
