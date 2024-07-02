package database

import (
	"context"
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var Store DBStorage

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
			Store.Cfg = cfg
			//Store.ConnectionPool, _ = pgxmock.NewPool()

			err := Store.Init()
			require.Error(t, err, tt.want.DBErr.Error())
			if err != nil {
				t.Skip(`connection refused`)
			}
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

func TestDBStorage_Ping(t *testing.T) {
	type want struct {
		ToDelete chan DeleteItem
		DBErr    DBError
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Ping`,
			want: want{
				ToDelete: make(chan DeleteItem),
				DBErr: DBError{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `Ping`,
					message:        `Connection pool is nil`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ping, err := Store.Ping(context.Background())
			require.Equal(t, false, ping)
			require.Error(t, err, tt.want.DBErr)
			if err != nil {
				t.Skip(`connection refused`)
			}
		})
	}
}

func TestDBStorage_Set(t *testing.T) {
	type want struct {
		ToDelete chan DeleteItem
		DBErr    DBError
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Set`,
			want: want{
				ToDelete: make(chan DeleteItem),
				DBErr: DBError{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `Set`,
					message:        `Connection pool is nil`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Store.Set(context.Background(), ``, ``, ``, 0)
			require.Error(t, err, tt.want.DBErr)
		})
	}
}

func TestDBStorage_Get(t *testing.T) {
	type want struct {
		ToDelete  chan DeleteItem
		DBErr     DBError
		data      string
		isDeleted bool
	}
	type args struct {
		link string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Get`,
			args: args{link: `asdasd`},
			want: want{
				ToDelete: make(chan DeleteItem),
				DBErr: DBError{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `Get`,
					message:        `Connection pool is nil`,
				},
				data:      ``,
				isDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, isDeleted, err := Store.Get(context.Background(), tt.args.link)
			require.Error(t, err, tt.want.DBErr)
			require.Equal(t, data, tt.want.data)
			require.Equal(t, isDeleted, tt.want.isDeleted)
		})
	}
}

func TestDBStorage_AsyncSaver(t *testing.T) {
	go func() {
		Store.AsyncSaver()
	}()

	select {
	case <-time.After(time.Millisecond * 100):
	case dataErr, ok := <-Store.AsyncSaverStatCh:
		require.Error(t, &dataErr)
		require.True(t, ok)
	}
}

func TestDBStorage_Destroy(_ *testing.T) {
	Store.Destroy()
}
