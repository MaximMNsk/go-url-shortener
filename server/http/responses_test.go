package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestOk(t *testing.T) {
	type want struct {
		status int
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: "Ok",
			want: want{
				status: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Ok(w)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)
			_ = result.Body.Close()
		})
	}
}

func TestConflict(t *testing.T) {
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
			name: "Conflict",
			want: want{
				status: http.StatusConflict,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Conflict(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)
			_ = result.Body.Close()
		})
	}
}

func TestCreatedJSON(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Created JSON",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusCreated,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			CreatedJSON(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestConflictJSON(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Conflict JSON",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusConflict,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ConflictJSON(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestOkAdditionalJSON(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Ok Additional JSON",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusOK,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			OkAdditionalJSON(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestNoContent(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "No Content",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusNoContent,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NoContent(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestUnauthorized(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Unauthorized",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusUnauthorized,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Unauthorized(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestAccepted(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Accepted",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusAccepted,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Accepted(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestGone(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Gone",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusGone,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Gone(w, tt.args.addData)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			bodyResult, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.want.addData, string(bodyResult))

			_ = result.Body.Close()
		})
	}
}

func TestForbidden(t *testing.T) {
	type args struct {
		addData Additional
	}

	type want struct {
		addData string
		status  int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Gone",
			args: args{
				addData: Additional{Place: `body`, InnerData: `Some text`},
			},
			want: want{
				status:  http.StatusForbidden,
				addData: `Some text`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Forbidden(w)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)

			_, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			_ = result.Body.Close()
		})
	}
}

func TestShutdown(t *testing.T) {
	type want struct {
		status int
	}

	tests := []struct {
		name string
		want want
	}{
		{
			name: "Shutdown",
			want: want{
				status: http.StatusServiceUnavailable,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Shutdown(w)
			result := w.Result()
			assert.Equal(t, tt.want.status, result.StatusCode)
			_ = result.Body.Close()
		})
	}
}
