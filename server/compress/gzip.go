package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func LengthHandle(w http.ResponseWriter, r *http.Request) (string, error) {
	// переменная reader будет равна r.Body или *gzip.Reader
	var reader io.Reader

	if r.Header.Get(`Content-Encoding`) == `gzip` {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return "", err
		}
		reader = gz
		defer gz.Close()
	} else {
		reader = r.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(len(body)), nil
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			next.ServeHTTP(w, r)
			return
		}

		//body, _ := io.ReadAll(r.Body)
		//fmt.Println(string(body))
		////gunzip, err := Decompress(body)
		////fmt.Println(gunzip)
		//
		//gzipw, err := gzip.NewWriterLevel(w, flate.BestCompression)
		//if err != nil {
		//	_, err := io.WriteString(w, err.Error())
		//	if err != nil {
		//		logger.PrintLog(logger.FATAL, "Can not write string")
		//		return
		//	}
		//	return
		//}
		//
		//defer gzipw.Close()
		//
		////bodyLen, _ := LengthHandle(w, r)
		//
		//w.Header().Set("Content-Encoding", "gzip")
		////w.Header().Set("Content-Length", bodyLen)
		//_, err = gzipw.Write(body)
		//if err != nil {
		//	return
		//}
		//
		//logger.PrintLog(logger.INFO, "Serve gzip...")
		//
		//next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gzipw}, r)
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
	defer r.Close()

	var b bytes.Buffer
	// в переменную b записываются распакованные данные
	_, err = b.ReadFrom(r)
	if err != nil {
		return nil, fmt.Errorf("failed decompress data: %v", err)
	}

	return b.Bytes(), nil
}
