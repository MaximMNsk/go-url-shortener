package compress

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func HandleInputValue(b []byte, needDecompress bool) ([]byte, error) {

	logger.PrintLog(logger.INFO, "HandleInputValue: "+string(b))
	logger.PrintLog(logger.INFO, "needDecompress: "+strconv.FormatBool(needDecompress))

	if needDecompress && b != nil {
		return Decompress(b)
	}
	return b, nil
}
func HandleOutputValue(b []byte, needCompress bool) ([]byte, error) {

	logger.PrintLog(logger.INFO, "HandleOutputValue: "+string(b))
	logger.PrintLog(logger.INFO, "needCompress: "+strconv.FormatBool(needCompress))

	if needCompress && b != nil {
		return Compress(b)
	} else {
		return b, nil
	}
}

type need struct {
	Compress   bool
	Decompress bool
}

type updatedWriter struct {
	http.ResponseWriter
	Compress need
}

func GzipHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		newWriter := updatedWriter{ResponseWriter: w}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}

		newWriter.Compress.Compress = false
		newWriter.Compress.Decompress = false

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			newWriter.Compress.Compress = true
			w.Header().Set("Content-Encoding", "gzip")
		}

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			newWriter.Compress.Decompress = true
		}

		logger.PrintLog(logger.INFO, r.RequestURI)
		logger.PrintLog(logger.INFO, r.Method)

		if r.Method == "POST" {
			if r.RequestURI == "/" {
				handlePOST(newWriter, r)
				return
			} else {
				handleAPI(newWriter, r)
				return
			}
		}

	})
}

// Compress сжимает слайс байт.
func Compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed init compress writer: %w", err)
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed compress data: %w", err)
	}
	// переменная b содержит сжатые данные
	return b.Bytes(), nil
}

// Decompress распаковывает слайс байт.
func Decompress(data []byte) ([]byte, error) {
	// переменная r будет читать входящие данные и распаковывать их
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed create reader: %w", err)
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
		return nil, fmt.Errorf("failed decompress data: %w", err)
	}

	return b.Bytes(), nil
}

/**
 * Обработка POST
 */
func handlePOST(res updatedWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	contentBody, errDecompress := HandleInputValue(contentBody, res.Compress.Decompress)
	if errDecompress != nil {
		logger.PrintLog(logger.ERROR, "Can not decompress data")
		httpResp.InternalError(res)
		return
	}

	linkFilePath := confModule.Config.Final.LinkFile

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkDataGet := files.JSONDataGet{}
	linkDataGet.Link = string(contentBody)
	// Проверяем, есть ли он (пока без валидаций).
	err := linkDataGet.Get(linkFilePath)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can not get link data")
		httpResp.InternalError(res)
		return
	}
	shortLink := linkDataGet.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkDataGet.ID == "" {
		linkDataSet := files.JSONDataSet{}
		linkDataSet.Link = string(contentBody)
		linkDataSet.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkDataSet.ID = linkID
		err := linkDataSet.Set(linkFilePath)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.InternalError(res)
			return
		}
		shortLink = linkDataSet.ShortLink
	}
	// Отдаем 201 ответ с шортлинком
	shortLinkByte := []byte(shortLink)
	shortLinkByte, err = HandleOutputValue(shortLinkByte, res.Compress.Compress)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can not compress data")
		httpResp.InternalError(res)
		return
	}
	additional := confModule.Additional{
		Place:     "body",
		InnerData: string(shortLinkByte),
	}
	httpResp.Created(res, additional)
}

type input struct {
	URL string `json:"url"`
}
type output struct {
	Result string `json:"result"`
}

func handleAPI(res updatedWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	contentBody, errDecompress := HandleInputValue(contentBody, res.Compress.Decompress)
	if errDecompress != nil {
		httpResp.InternalError(res)
		return
	}

	linkFilePath := confModule.Config.Final.LinkFile

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkDataGet := files.JSONDataGet{}
	var linkData input
	err := json.Unmarshal(contentBody, &linkData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkDataGet.Link = linkData.URL
	err = linkDataGet.Get(linkFilePath)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	shortLink := linkDataGet.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkDataGet.ID == "" {
		linkDataSet := files.JSONDataSet{}
		linkDataSet.Link = linkData.URL
		linkDataSet.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkDataSet.ID = linkID
		err := linkDataSet.Set(linkFilePath)
		if err != nil {
			httpResp.InternalError(res)
			return
		}
		shortLink = linkDataSet.ShortLink
	}
	var resp output
	resp.Result = shortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	JSONResp, err = HandleOutputValue(JSONResp, res.Compress.Compress)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	// Отдаем 201 ответ с шортлинком
	additional := confModule.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}
	httpResp.CreatedJSON(res, additional)
}
