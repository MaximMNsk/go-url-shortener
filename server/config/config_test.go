package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestHandleConfig(t *testing.T) {
	type args struct {
		envDB string
	}

	type wants struct {
		outputDBEnv     string
		outputDBDefault string
	}

	tests := []struct {
		name string
		args args
		want wants
	}{
		{
			name: `Test env`,
			args: args{envDB: `postgres://localhost`},
			want: wants{outputDBDefault: `postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable`, outputDBEnv: `postgres://localhost`},
		},
		{
			name: `Test default`,
			args: args{},
			want: wants{outputDBDefault: `postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == `Test env` {
				err := os.Setenv(`DATABASE_DSN`, tt.args.envDB)
				require.NoError(t, err)
				var cfg OuterConfig
				err = cfg.InitConfig(false)
				require.NoError(t, err)
				assert.Equal(t, tt.want.outputDBEnv, cfg.Final.DB)
			}
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != `Test default` {
				var cfg OuterConfig
				err := cfg.InitConfig(true)
				require.NoError(t, err)
				assert.Equal(t, tt.want.outputDBDefault, cfg.Final.DB)
			}
		})
	}
}
