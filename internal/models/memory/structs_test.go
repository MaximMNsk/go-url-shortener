package memory

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMemStorage_Init(t *testing.T) {
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
			name: `Test Init`,
			args: args{
				MemStorage{},
			},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.OuterConfig{}
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			storage := &MemStorage{}
			err = storage.Init(tt.args.Link, tt.args.ShortLink, tt.args.ID, tt.args.DeletedFlag, tt.args.Ctx, cfg)
			require.NoError(t, err)
		})
	}
}

func TestMemStorage_Ping(t *testing.T) {
	type args struct {
		MemStorage
	}
	type want struct {
		pingRes bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Init`,
			args: args{
				MemStorage{},
			},
			want: want{
				pingRes: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.OuterConfig{}
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			storage := &MemStorage{}
			err = storage.Init(tt.args.Link, tt.args.ShortLink, tt.args.ID, tt.args.DeletedFlag, tt.args.Ctx, cfg)
			require.NoError(t, err)
			res, err := storage.Ping()
			require.NoError(t, err)
			assert.Equal(t, tt.want.pingRes, res)
		})
	}
}

func TestMemStorage_Set(t *testing.T) {
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
					ID:          "ssssssssss",
					DeletedFlag: false,
				},
			},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.OuterConfig{}
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			storage := &MemStorage{}
			err = storage.Init(tt.args.Link, tt.args.ShortLink, tt.args.ID, tt.args.DeletedFlag, tt.args.Ctx, cfg)
			require.NoError(t, err)
			err = storage.Set()
			require.NoError(t, err)
		})
	}
}

func TestMemStorage_Get(t *testing.T) {
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
					ID:          "ssssssssss",
					DeletedFlag: false,
					Ctx:         context.Background(),
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
			cfg := config.OuterConfig{}
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			storage := &MemStorage{}
			err = storage.Init(tt.args.Link, tt.args.ShortLink, tt.args.ID, tt.args.DeletedFlag, tt.args.Ctx, cfg)
			require.NoError(t, err)
			err = storage.Set()
			require.NoError(t, err)
			URL, isDelete, err := storage.Get()
			require.NoError(t, err)
			assert.Equal(t, tt.want.URL, URL)
			assert.Equal(t, tt.want.isDelete, isDelete)
		})
	}
}

func TestMemStorage_Destroy(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: `Test get`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.OuterConfig{}
			err := cfg.InitConfig(true)
			require.NoError(t, err)
			storage := &MemStorage{}
			err = storage.Init(``, ``, ``, false, context.Background(), cfg)
			require.NoError(t, err)
			storage.Destroy()
		})
	}
}
