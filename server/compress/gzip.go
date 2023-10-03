package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
)

type GzipWriter struct {
	http.ResponseWriter
}

//func (w GzipWriter) Write(b []byte) (int, error) {
//	return w.Writer.Write(b)
//}

var needCompress = false

func HandleValue(b []byte) ([]byte, error) {
	fmt.Println(needCompress)
	if needCompress {
		return Compress(b)
	} else {
		return b, nil
	}
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !(strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") &&
			(strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "text/html"))) {
			next.ServeHTTP(w, r)
			return
		}

		needCompress = true

		w.Header().Set("Content-Encoding", "gzip")

		next.ServeHTTP(GzipWriter{ResponseWriter: w}, r)

		needCompress = false
	})
}

// Compress сжимает слайс байт.
func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	// создаём переменную w — в неё будут записываться входящие данные,
	// которые будут сжиматься и сохраняться в bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %v", err)
	}
	// запись данных
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %v", err)
	}
	// обязательно нужно вызвать метод Close() — в противном случае часть данных
	// может не записаться в буфер b; если нужно выгрузить все упакованные данные
	// в какой-то момент сжатия, используйте метод Flush()
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %v", err)
	}
	// переменная b содержит сжатые данные
	return b.Bytes(), nil
}

// Decompress распаковывает слайс байт.
func Decompress(data []byte) ([]byte, error) {
	// переменная r будет читать входящие данные и распаковывать их
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed create reader: %v", err)
	}

	defer func(r *gzip.Reader) {
		err := r.Close()
		if err != nil {
			return
		}
	}(r)

	var b bytes.Buffer
	// в переменную b записываются распакованные данные
	_, err = b.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %v", err)
	}

	return b.Bytes(), nil
}
