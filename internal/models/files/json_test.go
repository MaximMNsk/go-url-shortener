package files

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

var Store FileStorage

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
			Store = FileStorage{
				Cfg: cfg,
			}

			err := Store.Init()
			require.NoError(t, err)
		})
	}
}

func TestFileStorage_Ping(t *testing.T) {
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

			Store.Cfg = cfg
			//file.Cfg.Final.LinkFile = cfg.Default.LinkFile
			err := Store.Init()
			require.NoError(t, err)

			err = Store.Set(context.Background(), tt.args.link, tt.args.shortLink, tt.args.hashLink, 0)
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

			err := Store.Init()
			require.NoError(t, err)

			link, _, err := Store.Get(context.Background(), tt.args.hashLink)
			require.NoError(t, err)
			require.Equal(t, tt.want.link, link)
		})
	}
}

func TestErrorFile_Error(t *testing.T) {
	type args struct {
		layer          string
		parentFuncName string
		funcName       string
		message        string
	}
	type want struct {
		message string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Error`,
			args: args{
				layer:          `db`,
				parentFuncName: `some`,
				funcName:       `this`,
				message:        `word`,
			},
			want: want{
				message: `[db](some/this): word`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrorFile{
				layer:          tt.args.layer,
				funcName:       tt.args.funcName,
				message:        tt.args.message,
				parentFuncName: tt.args.parentFuncName,
			}
			msg := err.Error()
			assert.Equal(t, tt.want.message, msg)
		})
	}
}

func TestFileStorage_BatchSet(t *testing.T) {
	type args struct {
		URLs string
	}

	type want struct {
		res string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Error`,
			args: args{
				URLs: `[{"correlation_id":"aaa","original_url":"111"},{"correlation_id":"bbb","original_url":"222"}]`,
			},
			want: want{res: `[{"correlation_id":"aaa","short_url":"http://localhost:8080/aaa"},{"correlation_id":"bbb","short_url":"http://localhost:8080/bbb"}]`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := Store.BatchSet(context.Background(), []byte(tt.args.URLs), 0)
			require.NoError(t, err)
			require.Equal(t, tt.want.res, string(res))
		})
	}
}
