package db

import (
	"context"
	"testing"

	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {

	var Conf config.OuterConfig
	err := Conf.InitConfig(true)

	type args struct {
		connectString string
	}
	type want struct {
		pingResponse bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test length`,
			args: args{connectString: Conf.Final.DB},
			want: want{pingResponse: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
			}

			require.NoError(t, err)
			ctx := context.Background()
			pgPool, err := NewPool(ctx, Conf)
			require.NoError(t, err)
			defer pgPool.Close()
			err = pgPool.Ping(ctx)
			require.Error(t, err)
		})
	}

}

//func TestClose(t *testing.T) {
//	var pool *pgxpool.Pool
//	err := Close(pool)
//	require.Error(t, err, `DB is nil`)
//}
