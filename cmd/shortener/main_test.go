package main

import (
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_getShortURL(t *testing.T) {
	type args struct {
		linkID   string
		hostPort string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test link",
			args: args{linkID: "0X0X0X", hostPort: "http://localhost:8080"},
			want: "http://localhost:8080/0X0X0X",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getShortURL(tt.args.hostPort, tt.args.linkID), "getShortURL(%v)", tt.args.linkID)
		})
	}
}

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
	err := config.HandleConfig()
	if err != nil {
		return
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Set link" {
				bodyReader := strings.NewReader(tt.args.testLink)
				request := httptest.NewRequest(tt.args.method, tt.args.path, bodyReader)
				//request.Header.Set("Content-Type", tt.args.contentType)
				w := httptest.NewRecorder()
				handlePOST(w, request)

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

			if tt.name == "Get link" {
				request := httptest.NewRequest(tt.args.method, shortLink, nil)
				//request.Header.Set("Content-Type", tt.args.contentType)
				w := httptest.NewRecorder()
				handleGET(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				assert.Equal(t, result.Header.Get("Location"), tt.want.response)
				_ = result.Body.Close()
			}

		})
	}
}
