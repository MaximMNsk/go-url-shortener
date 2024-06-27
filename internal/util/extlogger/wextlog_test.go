package extlogger

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoggingResponseWriter_Write(t *testing.T) {
	var w httptest.ResponseRecorder
	data := &responseData{
		status: 0,
		size:   0,
	}
	lw := loggingResponseWriter{
		ResponseWriter: &w,
		responseData:   data,
	}
	write, err := lw.Write([]byte(`a`))
	require.NoError(t, err)
	require.Equal(t, 1, write)
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	var w httptest.ResponseRecorder
	data := &responseData{
		status: 0,
		size:   0,
	}
	lw := loggingResponseWriter{
		ResponseWriter: &w,
		responseData:   data,
	}
	lw.WriteHeader(200)
}

func TestLog(t *testing.T) {
	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(Log)
		r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	srv := http.Server{
		Handler: router,
		Addr:    `localhost:8088`,
	}
	go func() {
		err := srv.ListenAndServe()
		require.NoError(t, err)
	}()

	time.Sleep(time.Millisecond * 200)
	get, err := http.Get(`http://localhost:8088/`)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, get.StatusCode)
	err = get.Body.Close()
	require.NoError(t, err)

	go func() {
		time.Sleep(time.Millisecond * 400)
		err := srv.Shutdown(context.Background())
		require.NoError(t, err)
	}()
}
