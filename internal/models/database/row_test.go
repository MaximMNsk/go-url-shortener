package database

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDBStorage_Init(t *testing.T) {

	type want struct {
		ToDelete chan DeleteItem
		DBErr    DBError
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				ToDelete: make(chan DeleteItem),
				DBErr: DBError{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `prepare`,
					message:        `initialization error: dial tcp 127.0.0.1:5432: connect: connection refused`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config.OuterConfig
			ConfErr := cfg.InitConfig(true)
			require.NoError(t, ConfErr)
			var Store = DBStorage{
				Cfg: cfg,
			}

			err := Store.Init()
			require.Error(t, err, tt.want.DBErr.Error())
		})
	}
}

func TestExplodeURLs(t *testing.T) {
	type args struct {
		URLs string
	}
	type want struct {
		URLs []string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Explode URLs`,
			args: args{
				URLs: `["asd", "zxc"]`,
			},
			want: want{
				URLs: []string{`asd`, `zxc`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := explodeURLs(tt.args.URLs)
			require.NoError(t, err)
			var URLs []string

			err = json.Unmarshal([]byte(tt.args.URLs), &URLs)
			require.NoError(t, err)

			for _, v := range URLs {
				assert.Contains(t, result, v)
			}
		})
	}
}

func TestErrorDB_Error(t *testing.T) {
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
			err := DBError{
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
