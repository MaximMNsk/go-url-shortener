package cookie

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestBuildJWTString(t *testing.T) {
	type want struct {
	}

	type args struct {
		userID int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test BuildJWTString 0`,
			args: args{userID: 0},
			want: want{},
		},
		{
			name: `Test BuildJWTString 100`,
			args: args{userID: 100},
			want: want{},
		},
		{
			name: `Test BuildJWTString 999`,
			args: args{userID: 999},
			want: want{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildJWTString(tt.args.userID)
			require.NoError(t, err)
		})
	}
}

func TestParseJWTString(t *testing.T) {
	type want struct {
		userID int
	}
	type args struct {
		userID int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: `Test GetUserID 100`,
			args: args{userID: 100},
			want: want{userID: 100},
		},
		{
			name: `Test GetUserID 0`,
			args: args{userID: 0},
			want: want{userID: 0},
		},
		{
			name: `Test GetUserID 999`,
			args: args{userID: 999},
			want: want{userID: 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwt, err := BuildJWTString(tt.args.userID)
			uid := GetUserID(jwt)
			require.NoError(t, err)
			require.Equal(t, tt.want.userID, uid)
		})
	}
}

func TestAuthSetter(t *testing.T) {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(AuthSetter)
		r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	srv := http.Server{
		Handler: router,
		Addr:    `localhost:8089`,
	}
	go func() {
		err := srv.ListenAndServe()
		require.NoError(t, err)
	}()

	time.Sleep(time.Millisecond * 200)
	get, err := http.Get(`http://localhost:8089/`)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, get.StatusCode)

	go func() {
		time.Sleep(time.Millisecond * 400)
		err := srv.Shutdown(context.Background())
		require.NoError(t, err)
	}()
}
