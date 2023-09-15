package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

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
			err := os.Setenv("GO_APP", "D:\\Projects\\go-url-shortener")
			if err != nil {
				t.Error(err)
			}

			if tt.name == "Bad Request" {
				request := httptest.NewRequest(tt.args.method, tt.args.path, nil)
				w := httptest.NewRecorder()
				handleMainPage(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				return
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
				fmt.Println(shortLink)
				require.NotEmpty(t, shortLink)
				return
			}

			if tt.name == "Get link" {
				request := httptest.NewRequest(tt.args.method, shortLink, nil)
				w := httptest.NewRecorder()
				handleMainPage(w, request)

				result := w.Result()
				assert.Equal(t, tt.want.statusCode, result.StatusCode)
				assert.Contains(t, result.Header.Get("Content-Type"), tt.want.contentType)
				assert.Equal(t, result.Header.Get("Location"), tt.want.response)
				return
			}

		})
	}
}
