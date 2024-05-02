// Package server - слой обработчиков запросов от фронта.
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
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"io"
	"net/http"
	"net/http/pprof"
)

// ErrorHandlers - тип для работы с ошибками слоя обработчиков.
type ErrorHandlers struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

// Error - заменяет стандартный вызов метода своим.
func (e *ErrorHandlers) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

// HandleGET - метод получения URL из ShortURL.
func (s *Server) HandleGET(res http.ResponseWriter, req *http.Request) {

	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

	// Пришел ид.
	requestID := req.URL.Path[1:]

	s.Storage.Init(``, ``, requestID, false, req.Context(), s.Config)
	saved, deleted, err := s.Storage.Get()
	// 400.
	if err != nil {
		logger.PrintLog(logger.WARN, "Get exception: "+err.Error())
		httpResp.BadRequest(res)
		return
	}

	// 410.
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
		// Если есть, отдаем 307 редирект.
		//logger.PrintLog(logger.INFO, "Success")
		httpResp.TempRedirect(res, additional)
		return
	}

	// Если нет, отдаем BadRequest 400.
	logger.PrintLog(logger.WARN, "Not success")
	httpResp.BadRequest(res)
}

// HandlePOST - принимает запрос, наполняет объект данными, выполняет запрос к хранилищу.
func (s *Server) HandlePOST(res http.ResponseWriter, req *http.Request) {

	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

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
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkID)

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: shortLink,
	}

	s.Storage.Init(string(contentBody), shortLink, linkID, false, req.Context(), s.Config)
	setErr := s.Storage.Set()

	if setErr != nil {
		var pgErrType *pgconn.PgError
		if errors.As(setErr, &pgErrType) {
			if pgErrType.Code == pgerrcode.UniqueViolation {
				logger.PrintLog(logger.WARN, "Can not set link data: "+setErr.Error())
				httpResp.Conflict(res, additional)
				return
			}
		}
		logger.PrintLog(logger.ERROR, "Can not set link data: "+setErr.Error())
		httpResp.BadRequest(res)
		return
	}
	// Отдаем 201 ответ с шортлинком.
	httpResp.Created(res, additional)
}

type controllers map[string]bool

// HandleAPI - принимает и маршрутизирует API запросы.
// В зависимости от контроллера запроса и метода вызывает соответствующую функцию обработки.
func (s *Server) HandleAPI(res http.ResponseWriter, req *http.Request) {

	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

	ctrl := chi.URLParam(req, "query")

	availableCurls := make(controllers, 3)
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

// HandleAPIUserUrlsDelete - функция метода HandleAPI, агрегирующая логику удаления пакета URL через API.
// Работает для конкретного пользователя.
func HandleAPIUserUrlsDelete(res http.ResponseWriter, req *http.Request, s *Server) {
	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	httpResp.Accepted(res, httpResp.Additional{})

	s.Storage.Init(string(contentBody), ``, ``, false, req.Context(), s.Config)
	s.Storage.HandleUserUrlsDelete()
}

// HandleAPIUserUrls - функция метода HandleAPI, агрегирующая логику обработки пакета URL через API.
// Работает для конкретного пользователя.
func HandleAPIUserUrls(res http.ResponseWriter, req *http.Request, s *Server) {

	s.Storage.Init(``, ``, ``, false, req.Context(), s.Config)
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

// HandleAPIBatch - функция метода HandleAPI, агрегирующая логику обработки пакета URL через API.
func HandleAPIBatch(res http.ResponseWriter, req *http.Request, s *Server) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	s.Storage.Init(string(contentBody), ``, ``, false, req.Context(), s.Config)
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

// HandleAPIShorten - функция метода HandleAPI, агрегирующая логику обработки одного URL через API.
func HandleAPIShorten(res http.ResponseWriter, req *http.Request, s *Server) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	var apiData input
	err := json.Unmarshal(contentBody, &apiData)
	if err != nil {
		httpResp.BadRequest(res)
		return
	}
	linkID := sha1hash.Create(apiData.URL, 8)
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkID)

	var resp output
	resp.Result = shortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		httpResp.BadRequest(res)
		return
	}
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}

	s.Storage.Init(apiData.URL, shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkID), linkID, false, req.Context(), s.Config)
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

// HandlePing - метод сервера для проверки доступности хранилища.
func (s *Server) HandlePing(res http.ResponseWriter, req *http.Request) {
	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

	handlePingErr := &ErrorHandlers{
		layer:          `Handlers`,
		funcName:       `ChooseStorage`,
		parentFuncName: `-`,
	}

	s.Storage.Init(``, ``, ``, false, req.Context(), s.Config)
	ok, err := s.Storage.Ping()
	if ok {
		httpResp.Ok(res)
		return
	}
	logger.PrintLog(logger.ERROR, handlePingErr.Error()+`: `+err.Error())
	httpResp.InternalError(res)
}

// HandleOther - middleware для обработки неожидаемых запросов.
func HandleOther(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet || req.Method == http.MethodPost || req.Method == http.MethodDelete {
			next.ServeHTTP(res, req)
			return
		} else {
			httpResp.BadRequest(res)
			return
		}
	})
}

// ChooseStorage - метод выбора хранилища в зависимости от параметров конфигурации.
// Работает с конфигурацией, которую принимает вторым параметром.
// Возвращает модель для работы с хранилищем и ошибку.
func ChooseStorage(ctx context.Context, conf confModule.OuterConfig) (model.Storable, error) {
	var storage model.Storable
	if conf.Env.DB != "" || conf.Flag.DB != "" {
		pgPoolErr := &ErrorHandlers{
			layer:          `Handlers`,
			funcName:       `ChooseStorage`,
			parentFuncName: `-`,
		}
		pgPool, err := db.Connect(ctx, conf)
		if err != nil {
			pgPoolErr.message = `failed connect to DB`
			return &database.DBStorage{}, pgPoolErr
		}
		storage = &database.DBStorage{
			ConnectionPool: pgPool,
		}
		err = database.PrepareDB(conf.Final.DB)
		if err != nil {
			return storage, fmt.Errorf(pgPoolErr.Error()+`%w`, err)
		}

		go storage.AsyncSaver()
		return storage, nil
	}
	if conf.Env.LinkFile != `` || conf.Flag.LinkFile != `` {
		pgFileErr := &ErrorHandlers{
			layer:          `Handlers`,
			funcName:       `ChooseStorage`,
			parentFuncName: `-`,
		}

		storage = &files.FileStorage{}
		err := files.MakeStorageFile(conf.Final.LinkFile)
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

// Server - основная структура сервера, определяющая его работу.
type Server struct {
	Storage         model.Storable
	Routers         chi.Router
	Config          confModule.OuterConfig
	Context         context.Context
	HTTP            http.Server
	ShutdownProcess bool
}

// NewServ - создает новый сервер для тестов.
// Параметрами передаются конфигурация, модель сохранения, контекст для сервера.
func NewServ(c confModule.OuterConfig, s model.Storable, ctx context.Context) Server {
	return Server{Storage: s, Config: c, Context: ctx, ShutdownProcess: false}
}

// Start - запускает сервер в работу.
func (s *Server) Start() error {
	if s.ShutdownProcess {
		return nil
	}

	ctx := context.Background()

	storage, err := ChooseStorage(ctx, s.Config)
	if err != nil {
		var serverHandlersErr *ErrorHandlers
		if errors.As(err, &serverHandlersErr) {
			return fmt.Errorf(`can't handle storage: %w`, err)
		}
		return fmt.Errorf(`can't create storage environment: %w`, err)
	}

	s.Storage = storage

	s.Routers = chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(HandleOther)
	s.Routers.Route("/", func(r chi.Router) {
		s.Routers.Group(func(r chi.Router) {
			r.HandleFunc("/debug/pprof/*", pprof.Index)
			r.HandleFunc("/debug/pprof/profile", pprof.Profile)
			r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		})
		s.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthSetter)
			r.Post(`/`, s.HandlePOST)
			r.Post(`/api/{query}`, s.HandleAPI)
			r.Post(`/api/shorten/{query}`, s.HandleAPI)
			r.Get(`/ping`, s.HandlePing)
			r.Get(`/{query}`, s.HandleGET)
		})
		s.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthChecker)
			r.Delete(`/api/user/{query}`, s.HandleAPI)
			r.Get(`/api/user/{query}`, s.HandleAPI)
		})
	})

	s.HTTP.Addr = s.Config.Final.AppAddr
	s.HTTP.Handler = s.Routers
	err = s.HTTP.ListenAndServe()
	if err != nil {
		return fmt.Errorf(`can't start http listener: %w`, err)
	}

	return nil
}

// Stop - останавливает сервер.
func (s *Server) Stop() error {
	s.ShutdownProcess = true
	s.Storage.Destroy()
	err := s.HTTP.Shutdown(s.Context)
	if err != nil {
		return err
	}
	return nil
}
