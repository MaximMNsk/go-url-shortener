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
		DBErr    ErrorDB
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Init`,
			want: want{
				ToDelete: make(chan DeleteItem),
				DBErr: ErrorDB{
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

//func TestDBStorage_Ping(t *testing.T) {
//	type want struct {
//		pingRes bool
//	}
//
//	tests := []struct {
//		name string
//		want want
//	}{
//		{
//			name: `Test ping`,
//			want: want{pingRes: true},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			storage := mocks.Storable{}
//
//			storage.On(`Ping`).Return(true, nil)
//
//			res, err := storage.Ping()
//			require.NoError(t, err)
//			assert.Equal(t, tt.want.pingRes, res)
//		})
//	}
//}

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
			result, err := ExplodeURLs(tt.args.URLs)
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

//func TestDBStorage_Set(t *testing.T) {
//	type args struct {
//		link      string
//		shortLink string
//	}
//
//	tests := []struct {
//		name string
//		args args
//	}{
//		{
//			name: `Test Set`,
//			args: args{
//				link:      rand.StringBytes(10),
//				shortLink: rand.StringBytes(10),
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			storage := mocks.Storable{}
//			storage.Link = tt.args.link
//			storage.ShortLink = tt.args.shortLink
//			storage.ID = tt.args.shortLink
//
//			storage.On(`Set`).Return(nil)
//
//			err := storage.Set()
//			require.NoError(t, err)
//		})
//	}
//}
//
//func TestDBStorage_Get(t *testing.T) {
//	var Link, ShortLink = rand.StringBytes(10), rand.StringBytes(10)
//
//	type args struct {
//		link      string
//		shortLink string
//	}
//	type want struct {
//		link      string
//		isDeleted bool
//		err       error
//	}
//
//	tests := []struct {
//		name string
//		args args
//		want want
//	}{
//		{
//			name: `Test Get`,
//			args: args{
//				link:      Link,
//				shortLink: ShortLink,
//			},
//			want: want{
//				link:      Link,
//				isDeleted: false,
//				err:       nil,
//			},
//		},
//		{
//			name: `Test Get Wrong`,
//			args: args{
//				link:      ``,
//				shortLink: ``,
//			},
//			want: want{
//				link:      ``,
//				isDeleted: false,
//				err:       errors.ErrUnsupported,
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			storage := mocks.Storable{}
//			storage.Link = tt.args.link
//			storage.ShortLink = tt.args.shortLink
//			storage.ID = tt.args.shortLink
//
//			storage.On(`Get`).Return(tt.want.link, tt.want.isDeleted, tt.want.err)
//
//			link, isDeleted, err := storage.Get()
//			if tt.name == `Test Get` {
//				require.NoError(t, err)
//			}
//			assert.Equal(t, tt.want.link, link)
//			assert.Equal(t, tt.want.isDeleted, isDeleted)
//		})
//	}
//}

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
			err := ErrorDB{
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
