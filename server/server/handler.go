package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	model "github.com/MaximMNsk/go-url-shortener/internal/models/interface/models"
	"github.com/MaximMNsk/go-url-shortener/internal/models/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	memoryStorage "github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net/http"
)

type ErrorHandlers struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

func (e *ErrorHandlers) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

func (s *Server) HandleGET(res http.ResponseWriter, req *http.Request) {

	// Пришел ид
	requestID := req.URL.Path[1:]

	s.Storage.Init(``, ``, requestID, false, req.Context())
	saved, deleted, err := s.Storage.Get()
	// 400
	if err != nil {
		logger.PrintLog(logger.WARN, "Get exception: "+err.Error())
		httpResp.BadRequest(res)
		return
	}

	// 410
	if deleted {
		logger.PrintLog(logger.INFO, "Current item was deleted")
		httpResp.Gone(res, httpResp.Additional{})
		return
	}

	if saved != "" {
		additional := httpResp.Additional{
			Place:     "header",
			OuterData: "Location",
			InnerData: saved,
		}
		// Если есть, отдаем 307 редирект
		logger.PrintLog(logger.INFO, "Success")
		httpResp.TempRedirect(res, additional)
		return
	}

	// Если нет, отдаем BadRequest
	logger.PrintLog(logger.WARN, "Not success")
	httpResp.BadRequest(res)
}

/**
 * Обработка POST
 */
func (s *Server) HandlePOST(res http.ResponseWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			httpResp.BadRequest(res)
			return
		}
	}(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkID := sha1hash.Create(string(contentBody), 8)
	shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: shortLink,
	}

	s.Storage.Init(string(contentBody), shortLink, linkID, false, req.Context())
	setErr := s.Storage.Set()

	if setErr != nil {
		var pgErrType *pgconn.PgError
		if errors.As(setErr, &pgErrType) {
			if pgErrType.Code == `23505` {
				logger.PrintLog(logger.WARN, "Can not set link data: "+setErr.Error())
				httpResp.Conflict(res, additional)
				return
			}
		}
		logger.PrintLog(logger.ERROR, "Can not set link data: "+setErr.Error())
		httpResp.BadRequest(res)
		return
	}
	// Отдаем 201 ответ с шортлинком
	httpResp.Created(res, additional)
}

type controllers map[string]bool

func (s *Server) HandleAPI(res http.ResponseWriter, req *http.Request) {

	ctrl := chi.URLParam(req, "query")

	availableCurls := make(controllers)
	availableCurls["shorten"] = true
	availableCurls["batch"] = true
	availableCurls["urls"] = true

	if !availableCurls[ctrl] {
		httpResp.BadRequest(res)
		return
	}

	if ctrl == "shorten" {
		HandleAPIShorten(res, req, s)
		return
	}

	if ctrl == "batch" {
		HandleAPIBatch(res, req, s)
		return
	}

	if ctrl == `urls` && req.Method == `GET` {
		HandleAPIUserUrls(res, req, s)
		return
	}
	if ctrl == `urls` && req.Method == `DELETE` {
		HandleAPIUserUrlsDelete(res, req, s)
		return
	}
}

func HandleAPIUserUrlsDelete(res http.ResponseWriter, req *http.Request, s *Server) {
	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	httpResp.Accepted(res, httpResp.Additional{})

	s.Storage.Init(string(contentBody), ``, ``, false, req.Context())
	s.Storage.HandleUserUrlsDelete()
}

func HandleAPIUserUrls(res http.ResponseWriter, req *http.Request, s *Server) {

	s.Storage.Init(``, ``, ``, false, req.Context())
	byteRes, err := s.Storage.HandleUserUrls()

	if err != nil {
		httpResp.BadRequest(res)
		return
	}

	if byteRes == nil {
		httpResp.NoContent(res, httpResp.Additional{})
		return
	}

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(byteRes),
	}
	httpResp.OkAdditionalJSON(res, additional)
}

func HandleAPIBatch(res http.ResponseWriter, req *http.Request, s *Server) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	s.Storage.Init(string(contentBody), ``, ``, false, req.Context())
	resData, err := s.Storage.BatchSet()
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(resData),
	}

	if err != nil {
		var batchErr *pgconn.PgError
		if errors.As(err, &batchErr) {
			if batchErr.Code == `23505` {
				httpResp.ConflictJSON(res, additional)
				return
			}
			httpResp.BadRequest(res)
			return
		}
	}

	httpResp.CreatedJSON(res, additional)
}

type input struct {
	URL string `json:"url"`
}

type output struct {
	Result string `json:"result"`
}

func HandleAPIShorten(res http.ResponseWriter, req *http.Request, s *Server) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл, парсим его
	var apiData input
	err := json.Unmarshal(contentBody, &apiData)
	if err != nil {
		//httpResp.InternalError(res)
		httpResp.BadRequest(res)
		return
	}
	linkID := sha1hash.Create(apiData.URL, 8)
	shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)

	var resp output
	resp.Result = shortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		//httpResp.InternalError(res)
		httpResp.BadRequest(res)
		return
	}
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}

	s.Storage.Init(apiData.URL, shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID), linkID, false, req.Context())
	err = s.Storage.Set()

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == `23505` {
				logger.PrintLog(logger.WARN, "Can not set link data: "+err.Error())
				httpResp.ConflictJSON(res, additional)
				return
			}
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.BadRequest(res)
			return
		}
	}
	// Отдаем 201 ответ с шортлинком
	httpResp.CreatedJSON(res, additional)
}

func (s *Server) HandlePing(res http.ResponseWriter, req *http.Request) {
	handlePingErr := &ErrorHandlers{
		layer:          `Handlers`,
		funcName:       `ChooseStorage`,
		parentFuncName: `-`,
	}

	s.Storage.Init(``, ``, ``, false, req.Context())
	ok, err := s.Storage.Ping()
	if ok {
		httpResp.Ok(res)
		return
	}
	logger.PrintLog(logger.ERROR, handlePingErr.Error()+`: `+err.Error())
	httpResp.InternalError(res)
}

func HandleOther(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" || req.Method == "POST" || req.Method == "DELETE" {
			next.ServeHTTP(res, req)
		} else {
			httpResp.BadRequest(res)
			return
		}
	})
}

/**
 * Executor
 */

func ChooseStorage(ctx context.Context) (model.Storable, error) {
	var storage model.Storable
	if confModule.Config.Env.DB != "" || confModule.Config.Flag.DB != "" {
		pgPoolErr := &ErrorHandlers{
			layer:          `Handlers`,
			funcName:       `ChooseStorage`,
			parentFuncName: `-`,
		}
		pgPool, err := db.Connect(ctx)
		if err != nil {
			pgPoolErr.message = `failed connect to DB`
			return &database.DBStorage{}, pgPoolErr
		}
		storage = &database.DBStorage{
			ConnectionPool: pgPool,
		}
		err = database.PrepareDB()
		if err != nil {
			return storage, fmt.Errorf(pgPoolErr.Error()+`%w`, err)
		}

		go storage.AsyncSaver()
		return storage, nil
	}
	if confModule.Config.Env.LinkFile != `` || confModule.Config.Flag.LinkFile != `` {
		pgFileErr := &ErrorHandlers{
			layer:          `Handlers`,
			funcName:       `ChooseStorage`,
			parentFuncName: `-`,
		}

		storage = &files.FileStorage{}
		err := files.MakeStorageFile(confModule.Config.Final.LinkFile)
		if err != nil {
			pgFileErr.message = `can't init file storage`
			return storage, fmt.Errorf(pgFileErr.Error()+`: %w`, err)
		}
		return storage, nil
	}
	storage = &memory.MemStorage{
		Storage: memoryStorage.Storage{},
	}
	return storage, nil
}

type Server struct {
	Storage model.Storable
	Routers chi.Router
	Config  confModule.OuterConfig
	Context context.Context
}

func NewServ(c confModule.OuterConfig, s model.Storable, ctx context.Context) Server {
	return Server{Storage: s, Config: c, Context: ctx}
}
