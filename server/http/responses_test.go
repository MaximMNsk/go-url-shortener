package http

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBadRequest(t *testing.T) {
	type args struct {
		addData Additional
	}
	type want struct {
		status int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Bad request",
			args: args{},
			want: want{
				status: http.StatusBadRequest,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			BadRequest(w)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)
			_ = result.Body.Close()
		})
	}
}

func TestInternalError(t *testing.T) {
	type args struct {
		addData Additional
	}
	type want struct {
		status int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Internal Error",
			args: args{},
			want: want{
				status: http.StatusInternalServerError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			InternalError(w)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)
			_ = result.Body.Close()
		})
	}
}

func TestCreated(t *testing.T) {
	type args struct {
		addData Additional
		status  int
	}
	type headers struct {
		contentType string
		location    string
	}
	type want struct {
		status  int
		body    string
		headers headers
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Created",
			args: args{
				addData: Additional{
					Place:     "Body",
					OuterData: "",
					InnerData: "some value",
				},
				status: http.StatusCreated,
			},
			want: want{
				status: http.StatusCreated,
				body:   "some value",
			},
		},
		{
			name: "TempRedirect",
			args: args{
				addData: Additional{
					Place:     "Header",
					OuterData: "location",
					InnerData: "some value",
				},
				status: http.StatusTemporaryRedirect,
			},
			want: want{
				status: http.StatusTemporaryRedirect,
				headers: headers{
					location: "some value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			if tt.name == "Created" {
				Created(w, tt.args.addData)
			}
			if tt.name == "TempRedirect" {
				w.Header().Set(tt.args.addData.OuterData, tt.args.addData.InnerData)
				TempRedirect(w, tt.args.addData)
			}
			w.WriteHeader(tt.args.status)
			_, err := w.Write([]byte(tt.args.addData.InnerData))
			require.NoError(t, err)
			result := w.Result()

			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			if tt.name == "Created" {
				require.NotEmpty(t, bodyResult)
				assert.Equal(t, tt.want.body, string(bodyResult))
			}
			if tt.name == "TempRedirect" {
				assert.Equal(t, tt.want.headers.location, result.Header.Get("Location"))
			}
			_ = result.Body.Close()
		})
	}
}
