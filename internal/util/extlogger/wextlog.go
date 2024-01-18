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

type Header http.Header

type ResponseWriter interface {
	Header() Header
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
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}
func Log(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.New(os.Stdout).With().
			//Timestamp().
			Logger()

		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w, // встраиваем оригинальный http.ResponseWriter
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r) // внедряем реализацию http.ResponseWriter

		duration := time.Since(start).Seconds()
		scheme := ""
		if r.TLS == nil {
			scheme = "http://"
		} else {
			scheme = "https://"
		}

		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

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
	}
	return http.HandlerFunc(logFn)
}
