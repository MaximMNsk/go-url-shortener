package server

import (
	"context"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	random "github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/carlmjohnson/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

var Serv Server
var Cfg config.OuterConfig

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
			err := ErrorHandlers{
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

func TestChooseStorage(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test ChooseStorage`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Cfg.InitConfig(true)
			require.NoError(t, err)
			err = Cfg.InitConfig(false)
			require.NoError(t, err)
			_, err = ChooseStorage(context.Background(), Cfg)
			require.NoError(t, err)
		})
	}
}

func TestServer_Init(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Init`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Cfg.InitConfig(true)
			require.NoError(t, err)
			err = Serv.Init(Cfg, false)
			require.NoError(t, err)
		})
	}
}

func TestServer_Start(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Start`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				err := Serv.Start(context.Background())
				require.NoError(t, err)
			}()
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
		time.Sleep(500 * time.Millisecond)
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

func TestServer_Stop(t *testing.T) {
	type args struct{}
	type want struct{}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Stop`,
			args: args{},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			go func() {
				// Дождемся выполнения запросов
				time.Sleep(2000 * time.Millisecond)

				err := Serv.Stop(context.Background())
				require.NoError(t, err)
			}()
		})
	}
}

func ExampleServer_Init() {
	err := Serv.Init(Cfg, false)
	if err != nil {
		fmt.Printf("serv init err:%v\n", err)
	}
}

func ExampleServer_Start() {
	err := Serv.Start(context.Background())
	if err != nil {
		fmt.Printf("serv start err:%v\n", err)
	}
}

func ExampleHandleOther() {
	request, _ := http.NewRequest(http.MethodPut, `http://localhost:8080/`, nil)
	response, err := http.DefaultClient.Do(request)
	if err == nil {
		fmt.Println(response.StatusCode)
		_ = response.Body.Close()
	}
	// Output: 400
}

func ExampleServer_HandlePing() {
	request, _ := http.NewRequest(http.MethodGet, `http://localhost:8080/ping`, nil)
	response, err := http.DefaultClient.Do(request)
	if err == nil {
		fmt.Println(response.StatusCode)
		_ = response.Body.Close()
	}
	// Output: 200
}

func ExampleServer_HandlePOST() {
	body := strings.NewReader(`ya.ru`)
	request, _ := http.NewRequest(http.MethodPost, `http://localhost:8080/`, body)
	response, err := http.DefaultClient.Do(request)
	if err == nil {
		fmt.Println(response.StatusCode)
		_ = response.Body.Close()
	}
	// Output: 201
}

func ExampleServer_HandleGET() {
	shortLinkID := sha1hash.Create(`ya.ru`, 8)
	request, _ := http.NewRequest(http.MethodGet, `http://localhost:8080/`+shortLinkID, nil)
	client := http.DefaultClient
	client.CheckRedirect = requests.NoFollow
	response, err := client.Do(request)

	if err == nil {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Header.Get(`Location`))
		_ = response.Body.Close()
	}

	// Output:
	// 307
	// ya.ru
}

func ExampleServer_Stop() {
	err := Serv.Stop(context.Background())
	if err != nil {
		fmt.Printf("serv stop err:%v\n", err)
	}
}
