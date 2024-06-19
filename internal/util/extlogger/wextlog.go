// Package extlogger - middleware для перехвата и записи логов запроса.
package extlogger

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Header
//type Header http.Header

// ResponseWriter - интерфейс, который определяет структуру пакета.
type ResponseWriter interface {
	//Header() Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}
type (
	// берём структуру для хранения сведений об ответе
	responseData struct {
		status int
		size   int
	}

	// добавляем реализацию http.ResponseWriter
	loggingResponseWriter struct {
		// встраиваем оригинальный http.ResponseWriter
		http.ResponseWriter
		responseData *responseData
	}
)

// Write - записывает ответ, используя оригинальный ResponseWriter,
// возвращает размер записанных данных и ошибку.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	// захватываем размер
	r.responseData.size += size
	return size, err
}

// WriteHeader - записывает код статуса, используя оригинальный ResponseWriter
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	// захватываем код статуса
	r.responseData.status = statusCode
}

// Log - непосредственно, метод логирования на базе zerolog.
func Log(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.New(os.Stdout).With().
			Logger()

		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			// встраиваем оригинальный http.ResponseWriter
			ResponseWriter: w,
			responseData:   responseData,
		}
		// внедряем реализацию http.ResponseWriter
		h.ServeHTTP(&lw, r)

		duration := time.Since(start).Seconds()
		scheme := ""
		if r.TLS == nil {
			scheme = "http://"
		} else {
			scheme = "https://"
		}

		body, _ := io.ReadAll(r.Body)

		var UserID cookie.UserNum
		token, err := r.Cookie("token")
		if err == nil {
			UserID = cookie.UserNum(strconv.Itoa(cookie.GetUserID(token.Value)))
		}

		log.Info().
			Time("StartTime", start).
			Float64("Duration", duration).
			Str("Method", r.Method).
			Str("Content-Type", r.Header.Get("Content-Type")).
			Str("Accept-Encoding", r.Header.Get("Accept-Encoding")).
			Str("Content-Encoding", r.Header.Get("Content-Encoding")).
			Str("Body", string(body)).
			Str("URL", fmt.Sprintf("%s%s%s", scheme, r.Host, r.URL.Path)).
			Str("UserID", string(UserID)).
			Int("Status", responseData.status).
			Int("Size", responseData.size).
			Send()

		err = r.Body.Close()
		if err != nil {
			return
		}
	}
	return http.HandlerFunc(logFn)
}
