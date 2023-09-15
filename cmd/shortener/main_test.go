package main

import (
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
		linkID string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test link",
			args: args{linkID: "0X0X0X"},
			want: "http://localhost:8080/0X0X0X",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getShortURL(tt.args.linkID), "getShortURL(%v)", tt.args.linkID)
		})
	}
}

func Test_handleMainPage(t *testing.T) {
	type args struct {
		path     string
		method   string
		testLink string
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
			name: "Bad Request",
			args: args{
				method: http.MethodPut,
				path:   "http://localhost:8080/",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  400,
			},
		},
		{
			name: "Set link",
			args: args{
				method:   http.MethodPost,
				path:     "/",
				testLink: "https://ya.ru",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  201,
			},
		},
		{
			name: "Get link",
			args: args{
				method: http.MethodGet,
				path:   "/",
			},
			want: want{
				contentType: "text/plain",
				statusCode:  307,
				response:    "https://ya.ru",
			},
		},
	}
	var shortLink string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name == "Bad Request" {
				request := httptest.NewRequest(tt.args.method, tt.args.path, nil)
				w := httptest.NewRecorder()
				handleMainPage(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				_ = result.Body.Close()
			}

			if tt.name == "Set link" {
				bodyReader := strings.NewReader(tt.args.testLink)
				request := httptest.NewRequest(tt.args.method, tt.args.path, bodyReader)
				w := httptest.NewRecorder()
				handleMainPage(w, request)

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
				w := httptest.NewRecorder()
				handleMainPage(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				assert.Equal(t, result.Header.Get("Location"), tt.want.response)
				_ = result.Body.Close()
			}

		})
	}
}
