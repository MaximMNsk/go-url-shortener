package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConnect(t *testing.T) {

	var Conf config.OuterConfig
	Err := Conf.InitConfig(true)

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
			require.NoError(t, Err)
			ctx := context.Background()
			pgPool, err := Connect(ctx, Conf)
			require.NoError(t, err)
			defer pgPool.Close()
			err = pgPool.Ping(ctx)
			require.NoError(t, err)
		})
	}

}
