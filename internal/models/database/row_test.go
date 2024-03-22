package database

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/internal/util/randomizer"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

var Conf config.OuterConfig
var Store DBStorage

func TestDBStorage_Init(t *testing.T) {
	ConfErr := Conf.InitConfig(true)
	PgPool, PgErr := db.Connect(context.Background(), Conf)
	Store.ConnectionPool = PgPool

	type args struct {
		ctx       context.Context
		link      string
		shortLink string
		id        string
		isDeleted bool
	}
	type want struct {
		DBStorage
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Init`,
			args: args{
				ctx:       context.Background(),
				link:      `aaa`,
				shortLink: `bbb`,
				isDeleted: false,
			},
			want: want{DBStorage{
				Ctx:         context.Background(),
				Link:        `aaa`,
				ShortLink:   `bbb`,
				DeletedFlag: false,
				ToDeleteCh:  nil,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
				return
			}
			userNumber := cookie.UserNum(`UserID`)
			UserID, err := randomizer.RandDigitalBytes(3)
			require.NoError(t, err)
			ctx := context.WithValue(tt.args.ctx, userNumber, strconv.Itoa(UserID))
			require.NoError(t, ConfErr)
			require.NoError(t, PgErr)
			Store.Init(tt.args.link, tt.args.shortLink, tt.args.id, tt.args.isDeleted, ctx, Conf)
			assert.Equal(t, tt.want.DBStorage.Link, Store.Link)
			assert.Equal(t, tt.want.DBStorage.ShortLink, Store.ShortLink)
			assert.Equal(t, tt.want.DBStorage.DeletedFlag, Store.DeletedFlag)
		})
	}
}

func TestDBStorage_Ping(t *testing.T) {
	type args struct {
		storage DBStorage
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
			name: `Test ping`,
			args: args{
				storage: Store,
			},
			want: want{pingRes: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
				return
			}

			res, err := tt.args.storage.Ping()
			require.NoError(t, err)
			assert.Equal(t, tt.want.pingRes, res)
		})
	}
}

func TestPrepareDB(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test prepare DB`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
				return
			}

			err := PrepareDB(Conf.Final.DB)
			require.NoError(t, err)
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
			result, err := ExplodeURLs(tt.args.URLs)
			require.NoError(t, err)
			assert.Equal(t, tt.want.URLs, result)
		})
	}
}

var Link string
var ShortLink string

func TestDBStorage_Set(t *testing.T) {
	Link = rand.StringBytes(10)
	ShortLink = rand.StringBytes(10)

	type args struct {
		link      string
		shortLink string
	}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Set`,
			args: args{
				link:      Link,
				shortLink: ShortLink,
			},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
				return
			}

			Store.Link = tt.args.link
			Store.ShortLink = tt.args.shortLink
			Store.ID = tt.args.shortLink
			err := Store.Set()
			require.NoError(t, err)
		})
	}
}

func TestDBStorage_Get(t *testing.T) {
	type args struct {
		link      string
		shortLink string
	}
	type want struct {
		link      string
		isDeleted bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Get`,
			args: args{
				link:      Link,
				shortLink: ShortLink,
			},
			want: want{
				link:      Link,
				isDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if Conf.Env.DB == "" && Conf.Flag.DB == "" {
				assert.NotEmpty(t, Conf.Final.DB)
				return
			}

			Store.Link = tt.args.link
			Store.ShortLink = tt.args.shortLink
			Store.ID = tt.args.shortLink
			link, isDeleted, err := Store.Get()
			require.NoError(t, err)
			assert.Equal(t, tt.want.link, link)
			assert.Equal(t, tt.want.isDeleted, isDeleted)
		})
	}
}
