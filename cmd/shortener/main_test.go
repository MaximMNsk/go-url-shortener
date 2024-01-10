package main

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_handleMainPage(t *testing.T) {
	type args struct {
		path        string
		method      string
		testLink    string
		contentType string
	}
	type want struct {
		contentType string
		statusCode  int
		response    string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Set link",
			args: args{
				method:      http.MethodPost,
				path:        "http://localhost:8080/",
				testLink:    "https://ya.ru",
				contentType: "text/plain",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  201,
			},
		},
		{
			name: "Get link",
			args: args{
				method:      http.MethodGet,
				path:        "http://localhost:8080/",
				contentType: "text/plain",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  307,
				response:    "https://ya.ru",
			},
		},
	}
	var shortLink string
	config, _ := confModule.HandleConfig()
	storage := server.ChooseStorage()
	pgPool, _ := db.Connect(context.Background())
	serve := server.NewServ(config, storage, context.Background(), pgPool)

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Set link" {
				bodyReader := strings.NewReader(tt.args.testLink)
				request := httptest.NewRequest(tt.args.method, tt.args.path, bodyReader)
				w := httptest.NewRecorder()
				serve.HandlePOST(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				assert.Equal(t, result.Header.Get("Location"), tt.want.response)

				linkResult, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				shortLink = string(linkResult)
				require.NotEmpty(t, shortLink)
				_ = result.Body.Close()
			}

			//if tt.name == "Get link" {
			//	request := httptest.NewRequest(tt.args.method, shortLink, nil)
			//	w := httptest.NewRecorder()
			//	handleGET(w, request)
			//
			//	result := w.Result()
			//	assert.Equal(t, tt.want.statusCode, result.StatusCode)
			//	assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
			//	assert.Equal(t, tt.want.response, result.Header.Get("Location"))
			//	_ = result.Body.Close()
			//}

		})
	}
}
