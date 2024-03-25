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
			//err = os.Setenv(`DATABASE_DSN`, Cfg.Final.DB)
			//require.NoError(t, err)
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
			time.Sleep(2 * time.Second)
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
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)

			err = resp.Body.Close()
			require.NoError(t, err)
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
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)

			err = resp.Body.Close()
			require.NoError(t, err)
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
				link:   random.StringBytes(10),
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
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(request)

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)

			err = resp.Body.Close()
			require.NoError(t, err)
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
			require.NoError(t, err)

			cl := http.DefaultClient
			cl.CheckRedirect = requests.NoFollow
			resp, err := cl.Do(request)
			require.NoError(t, err)
			assert.Equal(t, tt.want.resp, resp.StatusCode)

			originalLink := resp.Header.Get(`Location`)
			assert.Equal(t, tt.want.link, originalLink)

			err = resp.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestServer_HandlePOST_GET(t *testing.T) {
	var link string
	var count = 10

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
			name: `Test Set`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodPost,
				link:   random.StringBytes(10),
			},
			want: want{
				resp: http.StatusCreated,
			},
		},
		{
			name: `Test Get`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodGet,
				link:   link,
			},
			want: want{
				resp: http.StatusTemporaryRedirect,
				link: link,
			},
		},
	}

	time.Sleep(100 * time.Millisecond)
	for i := 0; i <= count; i++ {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				if tt.name == `Test Set` {
					link = tt.args.link
					request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr, strings.NewReader(tt.args.link))
					require.NoError(t, err)

					resp, err := http.DefaultClient.Do(request)

					require.NoError(t, err)
					assert.Equal(t, tt.want.resp, resp.StatusCode)

					err = resp.Body.Close()
					require.NoError(t, err)
				}

				if tt.name == `Test Get` {
					time.Sleep(100 * time.Millisecond)
					shortLinkID := sha1hash.Create(link, 8)
					request, err := http.NewRequest(tt.args.method, `http://`+tt.args.addr+`/`+shortLinkID, nil)
					require.NoError(t, err)

					client := http.DefaultClient
					client.CheckRedirect = requests.NoFollow
					resp, err := client.Do(request)
					require.NoError(t, err)
					assert.Equal(t, tt.want.resp, resp.StatusCode)

					originalLink := resp.Header.Get(`Location`)
					assert.Equal(t, link, originalLink)

					err = resp.Body.Close()
					require.NoError(t, err)
				}
			})
		}
	}
}

//var links []string
//
//func BenchmarkServer_HandlePOST(b *testing.B) {
//	count := 5
//	_ = Cfg.InitConfig(true)
//
//	type Args struct {
//		addr   string
//		method string
//		link   string
//	}
//
//	Serv = NewServ(Cfg, Storage, context.Background())
//	Serv.Routers = chi.NewRouter().With(HandleOther)
//	Serv.Routers.Route(`/`, func(r chi.Router) {
//		r.Get(`/`, Serv.HandleGET)
//		r.Post(`/`, Serv.HandlePOST)
//		r.Get(`/{query}`, Serv.HandleGET)
//	})
//	srv := http.Server{Addr: Cfg.Final.AppAddr, Handler: Serv.Routers}
//
//	b.Run(`Get`, func(b *testing.B) {
//		go func() {
//			_ = srv.ListenAndServe()
//		}()
//		time.Sleep(1000 * time.Millisecond)
//		for i := 0; i <= count; i++ {
//			args := Args{
//				addr:   Cfg.Final.AppAddr,
//				method: http.MethodPost,
//				link:   random.RandStringBytes(10),
//			}
//
//			links = append(links, args.link)
//			request, _ := http.NewRequest(args.method, `http://`+args.addr, strings.NewReader(args.link))
//			resp, _ := http.DefaultClient.Do(request)
//			_ = resp.Body.Close()
//
//		}
//	})
//	time.Sleep(1000 * time.Millisecond)
//}
//
//func BenchmarkServer_HandleGET(b *testing.B) {
//	_ = Cfg.InitConfig(true)
//
//	type Args struct {
//		addr   string
//		method string
//		link   string
//	}
//
//	b.Run(`Get`, func(b *testing.B) {
//		for _, link := range links {
//			args := Args{
//				addr:   Cfg.Final.AppAddr,
//				method: http.MethodPost,
//				link:   random.RandStringBytes(10),
//			}
//
//			shortLinkID := sha1hash.Create(link, 8)
//			request, _ := http.NewRequest(args.method, `http://`+args.addr+`/`+shortLinkID, nil)
//
//			client := &http.Client{
//				CheckRedirect: func(req *http.Request, via []*http.Request) error {
//					return http.ErrUseLastResponse
//				},
//			}
//			resp, _ := client.Do(request)
//			_ = resp.Header.Get(`Location`)
//
//			_ = resp.Body.Close()
//		}
//	})
//}
