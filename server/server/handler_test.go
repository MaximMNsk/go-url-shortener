package server

import (
	"context"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/interface/models/mocks"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	random "github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var Serv Server
var Cfg config.OuterConfig
var Link string

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
		time.Sleep(100 * time.Millisecond)
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
				// Дождемся реального запуска
				time.Sleep(200 * time.Millisecond)

				err := Serv.Stop(context.Background())
				require.NoError(t, err)
			}()
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
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`Ping`, mock.Anything).Return(true, nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			request, err := http.NewRequestWithContext(context.Background(), tt.args.method, `http://`+tt.args.addr, nil)
			require.NoError(t, err)

			Serv.HandlePing(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)
		})
	}
}

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
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`Set`, mock.Anything, tt.args.link, mock.Anything, mock.Anything, mock.Anything).
				Return(nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			Link = tt.args.link
			request, err := http.NewRequestWithContext(context.Background(), tt.args.method, `http://`+tt.args.addr, strings.NewReader(tt.args.link))
			require.NoError(t, err)

			Serv.HandlePOST(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)
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
			shortLinkID := sha1hash.Create(tt.args.link, 8)

			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`Get`, mock.Anything, shortLinkID).
				Return(tt.args.link, false, nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			request, err := http.NewRequestWithContext(context.Background(), tt.args.method, `http://`+tt.args.addr+`/`+shortLinkID, nil)
			require.NoError(t, err)

			Serv.HandleGET(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)

			originalLink := resp.Header().Get(`Location`)
			assert.Equal(t, tt.want.link, originalLink)
		})
	}
}

func TestServer_HandleAPIBatch(t *testing.T) {
	type args struct {
		addr   string
		method string
		links  string
	}
	type want struct {
		resp  int
		links string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test Batch`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodPost,
				links:  `{"correlation_id": "abcabc","original_url": "http://ya.ru"}`,
			},
			want: want{
				resp:  http.StatusCreated,
				links: `{"correlation_id": "abcabc","short_url": "` + `http://` + Cfg.Final.AppAddr + `"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`BatchSet`, mock.Anything, []byte(tt.args.links), 0).
				Return([]byte(tt.want.links), nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			userNumber := cookie.UserNum(`UserID`)
			ctx := context.WithValue(context.Background(), userNumber, 0)
			request, err := http.NewRequestWithContext(
				ctx,
				tt.args.method,
				`http://`+tt.args.addr+`/api/shorten/batch`,
				strings.NewReader(tt.args.links),
			)
			require.NoError(t, err)

			Serv.HandleAPIBatch(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			originalLinks := string(body)
			assert.Equal(t, tt.want.links, originalLinks)
		})
	}
}

func TestServer_HandleAPIShorten(t *testing.T) {
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
			name: `Test Set API shorten`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodPost,
				link:   `{"url":"url"}`,
			},
			want: want{
				resp: http.StatusCreated,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`Set`, mock.Anything, `url`, mock.Anything, mock.Anything, 0).
				Return(nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			userNumber := cookie.UserNum(`UserID`)
			ctx := context.WithValue(context.Background(), userNumber, 0)
			request, err := http.NewRequestWithContext(
				ctx,
				tt.args.method,
				`http://`+tt.args.addr+`/api/shorten`,
				strings.NewReader(tt.args.link),
			)
			require.NoError(t, err)

			Serv.HandleAPIShorten(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)
		})
	}
}

func TestServer_HandleAPIUserUrls(t *testing.T) {
	type args struct {
		addr   string
		method string
		userID int
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
			name: `Test APIUserUrls`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodGet,
				userID: 0,
			},
			want: want{
				resp: http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`HandleUserUrls`, mock.Anything, 0).Return([]byte(`321`), nil)
			Serv.Storage = storageMock

			userNumber := cookie.UserNum(`UserID`)
			ctx := context.WithValue(context.Background(), userNumber, tt.args.userID)
			request, _ := http.NewRequestWithContext(ctx, http.MethodPost, tt.args.addr+`/api/user/urls`, nil)
			resp := httptest.NewRecorder()
			Serv.HandleAPIUserUrls(resp, request)
			require.Equal(t, tt.want.resp, resp.Code)
		})
	}
}

func TestServer_HandleAPIUserUrlsDelete(t *testing.T) {
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
			name: `Test User Urls Delete`,
			args: args{
				addr:   Cfg.Final.AppAddr,
				method: http.MethodDelete,
				link:   random.StringBytes(10),
			},
			want: want{
				resp: http.StatusAccepted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`HandleUserUrlsDelete`, mock.Anything, 0).
				Return(nil)
			Serv.Storage = storageMock
			resp := httptest.NewRecorder()

			userNumber := cookie.UserNum(`UserID`)
			ctx := context.WithValue(context.Background(), userNumber, 0)
			request, err := http.NewRequestWithContext(ctx, tt.args.method, `http://`+tt.args.addr+`/api/user/urls`, strings.NewReader(tt.args.link))
			require.NoError(t, err)

			Serv.HandleAPIUserUrlsDelete(resp, request)
			assert.Equal(t, tt.want.resp, resp.Code)
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
	go func() {
		err := Serv.Start(context.Background())
		if err != nil {
			fmt.Printf("serv start err:%v\n", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
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

func ExampleServer_Stop() {
	go func() {
		err := Serv.Stop(context.Background())
		if err != nil {
			fmt.Printf("serv stop err:%v\n", err)
		}
	}()
}

func ExampleServer_HandlePing() {
	var storageMock mocks.Storable
	storageMock.
		On(`Ping`, mock.Anything).Return(true, nil)
	Serv.Storage = &storageMock
	resp := httptest.NewRecorder()

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, `http://localhost:8080/ping`, nil)

	Serv.HandlePing(resp, request)
	fmt.Println(resp.Code)

	// Output: 200
}

func ExampleServer_HandlePOST() {
	var storageMock mocks.Storable
	storageMock.
		On(`Set`, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	Serv.Storage = &storageMock
	resp := httptest.NewRecorder()

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, `http://localhost:8080/`, strings.NewReader(`ya.ru`))

	Serv.HandlePOST(resp, request)
	fmt.Println(resp.Code)

	// Output: 201
}

func ExampleServer_HandleGET() {
	link := `ya.ru`
	shortLinkID := sha1hash.Create(link, 8)
	var storageMock mocks.Storable
	storageMock.
		On(`Get`, mock.Anything, shortLinkID).
		Return(link, false, nil)
	Serv.Storage = &storageMock
	resp := httptest.NewRecorder()

	request, _ := http.NewRequest(http.MethodGet, `http://localhost:8080/`+shortLinkID, nil)
	Serv.HandleGET(resp, request)
	fmt.Println(resp.Code)
	fmt.Println(resp.Header().Get("Location"))

	// Output:
	// 307
	// ya.ru
}
