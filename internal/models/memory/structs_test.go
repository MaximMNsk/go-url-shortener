package memory

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var Conf config.OuterConfig
var ConfErr error
var Unit MemStorage

func TestMemStorage_Set(t *testing.T) {
	ConfErr = Conf.InitConfig(true)

	type args struct {
		MemStorage
	}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test set`,
			args: args{
				MemStorage{
					Link:        "ya.ru",
					ShortLink:   "ssssssssss",
					ID:          "123",
					DeletedFlag: false,
					Ctx:         context.Background(),
					Cfg:         Conf,
				},
			},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Unit.Init(tt.args.Link, tt.args.ShortLink, tt.args.ID, tt.args.DeletedFlag, tt.args.Ctx, tt.args.Cfg)
			err := Unit.Set()
			require.NoError(t, err)
			require.NoError(t, ConfErr)
		})
	}
}

func TestMemStorage_Get(t *testing.T) {
	ConfErr = Conf.InitConfig(true)

	type args struct {
		MemStorage
	}
	type want struct {
		URL      string
		isDelete bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test get`,
			args: args{
				MemStorage{
					Link:        "ya.ru",
					ShortLink:   "ssssssssss",
					ID:          "123",
					DeletedFlag: false,
					Ctx:         context.Background(),
					Cfg:         Conf,
				},
			},
			want: want{
				URL:      `ya.ru`,
				isDelete: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			URL, isDelete, err := Unit.Get()
			require.NoError(t, err)
			assert.Equal(t, tt.want.URL, URL)
			assert.Equal(t, tt.want.isDelete, isDelete)
		})
	}
}
