// Package compress - обработчик запросов сжатием.
// В зависимости от данных в заголовке запроса сжимает или разжимает как запрос, так и ответ.
package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
)

type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header - переопределен стандартный метод через gzip.
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write - переопределен стандартный метод через gzip.
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader - переопределен стандартный метод через gzip.
func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
	c.zw.Reset(c.w)
}

// Close - переопределен стандартный метод через gzip.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	zr.Multistream(false)

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read - переопределен стандартный метод через gzip.
func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close - переопределен стандартный метод через gzip.
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// GzipHandler - функция middleware, который, в зависимости от информации в заголовках сжимает или разархивирует данные.
func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}

		ow := w
		//var ow http.ResponseWriter

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newCompressWriter(w)
			cw.Header().Set("Content-Encoding", "gzip")
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer func(cw *compressWriter) {
				err := cw.Close()
				if err != nil {
					logger.PrintLog(logger.ERROR, "Can't close compress writer: "+err.Error(), true)
				}
			}(cw)
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				logger.PrintLog(logger.ERROR, err.Error(), true)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer func(cr *compressReader) {
				err := cr.Close()
				if err != nil {
					logger.PrintLog(logger.ERROR, "Can't close compress reader", true)
				}
			}(cr)
		}

		// передаём управление хендлеру
		next.ServeHTTP(ow, r)
	})
}
