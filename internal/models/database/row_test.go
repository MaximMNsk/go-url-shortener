package database

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/MaximMNsk/go-url-shortener/server/config"
	_ "github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			var storage DBStorage
			var cfg config.OuterConfig
			ConfErr := cfg.InitConfig(true)
			require.NoError(t, ConfErr)
			storage.Cfg = cfg

			err := storage.Init()
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
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()
			storage := &DBStorage{ConnectionPool: mockPool}
			mockPool.ExpectPing().WillReturnError(nil)

			ping, err := storage.Ping(context.Background())
			require.Equal(t, true, ping)
			require.NoError(t, err)
		})
	}
}

func TestDBStorage_Set(t *testing.T) {
	type want struct {
		DBErr DBError
	}
	tests := []struct {
		name string
		want want
	}{
		{
			name: `Test Set`,
			want: want{
				DBErr: DBError{
					layer:          layer,
					parentFuncName: ``,
					funcName:       `Set`,
					message:        ``,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()
			storage := DBStorage{ConnectionPool: mockPool}

			commandTag := pgconn.NewCommandTag("INSERT 0 1")
			mockPool.
				ExpectExec(`insert into public.short_links`).
				WithArgs(`ya.ru`, `http://localhost:8080/123`, `123`, 0).
				WillReturnResult(commandTag)

			err = storage.Set(
				context.Background(),
				`ya.ru`,
				`http://localhost:8080/123`,
				`123`,
				0)
			require.NoError(t, err)
		})
	}
}

func TestDBStorage_Get(t *testing.T) {
	type want struct {
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
				data:      `ya.ru`,
				isDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()
			storage := &DBStorage{ConnectionPool: mockPool}

			rows := mockPool.NewRows([]string{"original_url", "is_deleted"}).
				AddRow(tt.want.data, false)
			mockPool.ExpectQuery(`select original_url, is_deleted from public.short_links`).WithArgs(tt.args.link).WillReturnRows(rows)

			data, isDeleted, err := storage.Get(context.Background(), tt.args.link)
			require.NoError(t, err)
			require.Equal(t, data, tt.want.data)
			require.Equal(t, isDeleted, tt.want.isDeleted)
		})
	}
}

func TestDBStorage_BatchSet(t *testing.T) {
	cfg := config.OuterConfig{}
	err := cfg.InitConfig(true)
	require.NoError(t, err)

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	storage := DBStorage{
		ToDeleteCh:       make(chan DeleteItem),
		AsyncSaverStatCh: make(chan DBError),
		Cfg:              cfg,
		ConnectionPool:   mockPool,
	}

	commandTag := pgconn.NewCommandTag("INSERT 0 1")
	mockPool.
		ExpectBatch().
		ExpectExec(`insert into public.short_links `).
		WithArgs(`ya.ru`, `http://localhost:8080/123`, `123`, 0).
		WillReturnResult(commandTag)

	data := []byte(`[{"correlation_id": "123", "original_url": "ya.ru"}]`)
	set, err := storage.BatchSet(context.Background(), data, 0)

	res := []byte(`[{"correlation_id":"123","short_url":"http://localhost:8080/123"}]`)

	require.NoError(t, err)
	require.Equal(t, res, set)
}

func TestDBStorage_BatchUpdate(t *testing.T) {
	cfg := config.OuterConfig{}
	err := cfg.InitConfig(true)
	require.NoError(t, err)

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	storage := DBStorage{
		ToDeleteCh:       make(chan DeleteItem),
		AsyncSaverStatCh: make(chan DBError),
		Cfg:              cfg,
		ConnectionPool:   mockPool,
	}

	commandTag := pgconn.NewCommandTag("INSERT 0 1")
	mockPool.
		ExpectBatch().
		ExpectExec(`update public.short_links set is_deleted = true where uid`).
		WithArgs(`asd`).
		WillReturnResult(commandTag)
	err = storage.BatchUpdate(context.Background(), `["asd"]`, 0)
	require.NoError(t, err)
}

func TestDBStorage_HandleUserUrls(t *testing.T) {
	cfg := config.OuterConfig{}
	err := cfg.InitConfig(true)
	require.NoError(t, err)

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	storage := DBStorage{
		ToDeleteCh:       make(chan DeleteItem),
		AsyncSaverStatCh: make(chan DBError),
		Cfg:              cfg,
		ConnectionPool:   mockPool,
	}

	rows := mockPool.NewRows([]string{"original_url", "short_url"}).
		AddRow(`ya.ru`, `http://localhost:8080/123`)
	mockPool.
		ExpectQuery(`select original_url, short_url from public.short_links`).
		WithArgs(0).WillReturnRows(rows)

	res, err := storage.HandleUserUrls(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, res, []byte(`[{"original_url":"ya.ru","short_url":"http://localhost:8080/123"}]`))
}

func TestDBStorage_HandleUserUrlsDelete(t *testing.T) {

}

func TestDBStorage_AsyncSaver(t *testing.T) {
	cfg := config.OuterConfig{}
	err := cfg.InitConfig(true)
	require.NoError(t, err)

	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	commandTag := pgconn.NewCommandTag("INSERT 0 1")
	mockPool.
		ExpectBatch().
		ExpectExec(`update public.short_links set is_deleted = true where uid`).
		WithArgs(`6qxTVvsy`).
		WillReturnResult(commandTag)

	storage := DBStorage{
		ToDeleteCh:       make(chan DeleteItem),
		AsyncSaverStatCh: make(chan DBError),
		Cfg:              cfg,
		ConnectionPool:   mockPool,
	}

	go func() {
		storage.AsyncSaver()
	}()

	storage.ToDeleteCh <- DeleteItem{
		URLs:   `["6qxTVvsy"]`,
		UserID: 0,
	}

	select {
	case <-time.After(time.Millisecond * 100):
	case dataErr, ok := <-storage.AsyncSaverStatCh:
		require.NoError(t, &dataErr)
		require.True(t, ok)
	}
}

func TestDBStorage_Destroy(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	storage := DBStorage{
		ConnectionPool: mockPool,
	}

	mockPool.ExpectClose().WillReturnError(nil)

	err = storage.Destroy()
	require.NoError(t, err)
}
