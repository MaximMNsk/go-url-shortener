package files

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestFileStorage_Init(t *testing.T) {
	type want struct {
		DBErr ErrorFile
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				DBErr: ErrorFile{
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
			var Store = FileStorage{
				Cfg: cfg,
			}

			err := Store.Init()
			require.NoError(t, err)
		})
	}
}

func TestFileStorage_Set(t *testing.T) {

	var cfg config.OuterConfig
	errCfg := cfg.InitConfig(true)

	type args struct {
		fileName  string
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
				fileName:  filepath.Join(cfg.Default.LinkFile),
				link:      `aaa`,
				shortLink: `http://localhost:8080:bbb`,
				hashLink:  `bbb`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, errCfg)

			var file FileStorage
			file.Cfg = cfg
			//file.Cfg.Final.LinkFile = cfg.Default.LinkFile
			err := file.Init()
			require.NoError(t, err)

			err = file.Set(context.Background(), tt.args.link, tt.args.shortLink, tt.args.hashLink, 0)
			require.NoError(t, err)
			require.FileExists(t, filepath.Join(tt.args.fileName))
		})
	}
}

func TestFileStorage_Get(t *testing.T) {

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
				fileName: filepath.Join(cfg.Default.LinkFile),
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
			assert.FileExists(t, tt.args.fileName)
			require.NoError(t, errCfg)

			file := FileStorage{
				Cfg: cfg,
			}
			err := file.Init()
			require.NoError(t, err)

			link, _, err := file.Get(context.Background(), tt.args.hashLink)
			require.NoError(t, err)
			require.Equal(t, tt.want.link, link)
		})
	}
}
