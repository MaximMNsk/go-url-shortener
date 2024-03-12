package server

import (
	"context"
	model "github.com/MaximMNsk/go-url-shortener/internal/models/interface/models"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	random "github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/carlmjohnson/requests"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var Serv Server
var Cfg config.OuterConfig
var Storage model.Storable

func TestChooseStorage(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test NewServ`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Cfg.InitConfig(true)
			require.NoError(t, err)
			err = os.Setenv(`DATABASE_DSN`, Cfg.Final.DB)
			require.NoError(t, err)
			err = Cfg.InitConfig(false)
			require.NoError(t, err)
			Storage, err = ChooseStorage(context.Background(), Cfg)
			require.NoError(t, err)
		})
	}
}

func TestNewServ(t *testing.T) {
	type args struct {
		addr string
	}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test NewServ`,
			args: args{
				addr: Cfg.Final.AppAddr,
			},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Serv = NewServ(Cfg, Storage, context.Background())
			Serv.Routers = chi.NewRouter().With(HandleOther)
			Serv.Routers.Route(`/`, func(r chi.Router) {
				r.Get(`/ping`, Serv.HandlePing)
				r.Get(`/`, Serv.HandleGET)
				r.Post(`/`, Serv.HandlePOST)
				r.Get(`/{query}`, Serv.HandleGET)
			})
			srv := http.Server{Addr: tt.args.addr, Handler: Serv.Routers}
			go func() {
				_ = srv.ListenAndServe()
			}()
			time.Sleep(1000 * time.Millisecond)
		})
	}
}

func TestHandleOther(t *testing.T) {
	type args struct {
		addr   string
		method string
	}
	type want struct {
		resp int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Put`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodPut,
			},
			want: want{
				resp: http.StatusBadRequest,
			},
		},
		{
			name: `Test Head`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodHead,
			},
			want: want{
				resp: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr, nil)
			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)
		})
	}
}

func TestServer_HandlePing(t *testing.T) {
	type args struct {
		addr   string
		method string
	}
	type want struct {
		resp int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Ping`,
			args: args{
				addr:   Cfg.Final.AppAddr + `/ping`,
				method: http.MethodGet,
			},
			want: want{
				resp: http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr, nil)
			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)
		})
	}
}

var Link string

func TestServer_HandlePOST(t *testing.T) {
	type args struct {
		addr   string
		method string
		link   string
	}
	type want struct {
		resp int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Set`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodPost,
				link:   random.RandStringBytes(10),
			},
			want: want{
				resp: http.StatusCreated,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Link = tt.args.link
			request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr, strings.NewReader(tt.args.link))
			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)
		})
	}
}

func TestServer_HandleGET(t *testing.T) {
	type args struct {
		addr   string
		method string
		link   string
	}
	type want struct {
		resp int
		link string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Get`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodGet,
				link:   Link,
			},
			want: want{
				resp: http.StatusTemporaryRedirect,
				link: Link,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(100 * time.Millisecond)
			shortLinkID := sha1hash.Create(tt.args.link, 8)
			request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr+`/`+shortLinkID, nil)
			cl := http.DefaultClient
			cl.CheckRedirect = requests.NoFollow
			resp, err := cl.Do(request)
			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)

			originalLink := resp.Header.Get(`Location`)
			assert.Equal(t, tt.want.link, originalLink)
		})
	}
}
