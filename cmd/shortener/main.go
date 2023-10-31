package main

import (
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/interface/model"
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/models/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	memoryStorage "github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net/http"
)

/**
 * Request type handlers
 */

func (s *Server) handleGET(res http.ResponseWriter, req *http.Request) {

	// Пришел ид
	requestID := req.URL.Path[1:]

	s.storage.Init(``, ``, requestID, req.Context())
	saved, err := s.storage.Get()
	if err != nil {
		logger.PrintLog(logger.WARN, "Get exception: "+err.Error())
		httpResp.BadRequest(res)
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
func (s *Server) handlePOST(res http.ResponseWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
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

	s.storage.Init(string(contentBody), shortLink, linkID, req.Context())
	err := s.storage.Set()

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	if pgErr != nil {
		if pgErr.SQLState() == `23505` {
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.Conflict(res, additional)
			return
		}
	}

	if err != nil {
		logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
		httpResp.InternalError(res)
		return
	}

	// Отдаем 201 ответ с шортлинком
	httpResp.Created(res, additional)
}

type controllers map[string]bool

func (s *Server) handleAPI(res http.ResponseWriter, req *http.Request) {

	ctrl := chi.URLParam(req, "query")

	availableCurls := make(controllers)
	availableCurls["shorten"] = true
	availableCurls["batch"] = true

	if !availableCurls[ctrl] {
		httpResp.BadRequest(res)
		return
	}

	if ctrl == "shorten" {
		handleAPIShorten(res, req, s)
		return
	}

	if ctrl == "batch" {
		handleAPIBatch(res, req, s)
		return
	}
}

func handleAPIBatch(res http.ResponseWriter, req *http.Request, s *Server) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	s.storage.Init(string(contentBody), ``, ``, req.Context())
	resData, err := s.storage.BatchSet()

	var batchErr *pgconn.PgError
	errors.As(err, &batchErr)

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(resData),
	}

	if batchErr != nil {
		if batchErr.SQLState() == `23505` {
			httpResp.ConflictJSON(res, additional)
			return
		} else {
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

func handleAPIShorten(res http.ResponseWriter, req *http.Request, s *Server) {

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
		httpResp.InternalError(res)
		return
	}
	linkID := sha1hash.Create(apiData.URL, 8)
	shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)

	var resp output
	resp.Result = shortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}

	s.storage.Init(apiData.URL, shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID), linkID, req.Context())
	err = s.storage.Set()

	var pgErr *pgconn.PgError
	errors.As(err, &pgErr)

	if pgErr != nil {
		if pgErr.SQLState() == `23505` {
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.ConflictJSON(res, additional)
			return
		}
	}

	if err != nil {
		httpResp.InternalError(res)
		return
	}

	// Отдаем 201 ответ с шортлинком
	httpResp.CreatedJSON(res, additional)
}

func (s *Server) handlePing(res http.ResponseWriter, req *http.Request) {

	if db.GetDB() == nil {
		err := db.Connect()
		defer func() {
			_ = db.Close()
		}()
		if err != nil {
			logger.PrintLog(logger.ERROR, err.Error())
			httpResp.InternalError(res)
			return
		}
	}
	httpResp.Ok(res)
}

func handleOther(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" || req.Method == "POST" {
			next.ServeHTTP(res, req)
		} else {
			httpResp.BadRequest(res)
		}
	})
}

/**
 * Executor
 */

func initStorage() model.Storable {
	var storage model.Storable
	if confModule.Config.Env.DB != "" || confModule.Config.Flag.DB != "" {
		storage = &database.DBStorage{}
		return storage
	}
	if confModule.Config.Env.LinkFile != `` || confModule.Config.Flag.LinkFile != `` {
		storage = &files.FileStorage{}
		return storage
	}
	storage = &memory.MemStorage{
		Storage: memoryStorage.Storage{},
	}
	return storage
}

type Server struct {
	storage model.Storable
	routers chi.Router
	config  confModule.OuterConfig
}

func NewServ(c confModule.OuterConfig, s model.Storable) Server {
	return Server{storage: s, config: c}
}

func main() {

	logger.PrintLog(logger.INFO, "Start server")
	logger.PrintLog(logger.INFO, "Handle config")

	conf, err := confModule.HandleConfig()
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
	}

	if confModule.Config.Env.DB != `` || confModule.Config.Flag.DB != `` {
		err := db.Connect()
		if err != nil {
			logger.PrintLog(logger.ERROR, "Failed connect to DB")
		}
		database.PrepareDB(db.GetDB())
		defer func() {
			err := db.Close()
			if err != nil {
				logger.PrintLog(logger.ERROR, "Failed close connection to DB")
			}
		}()
	}

	storage := initStorage()
	server := NewServ(conf, storage)

	logger.PrintLog(logger.INFO, "Declaring router")

	server.routers = chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(handleOther)
	server.routers.Route("/", func(r chi.Router) {
		r.Post(`/`, server.handlePOST)
		r.Post(`/api/{query}`, server.handleAPI)
		r.Post(`/api/shorten/{query}`, server.handleAPI)
		r.Get(`/ping`, server.handlePing)
		r.Get(`/{query}`, server.handleGET)
	})

	logger.PrintLog(logger.INFO, "Starting server")

	err = http.ListenAndServe(confModule.Config.Final.AppAddr, server.routers)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
