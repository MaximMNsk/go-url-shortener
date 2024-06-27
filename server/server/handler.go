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
// Эндпоинт с методом GET и путём /{id}, где id — идентификатор сокращённого URL (например, /EwHXdJfB).
// В случае успешной обработки запроса сервер возвращает ответ с кодом 307 и оригинальным URL в HTTP-заголовке Location.
func (s *Server) HandleGET(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandleGET`, s.LogEnabled)

	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

	// Пришел ид.
	linkHash := req.URL.Path[1:]

	saved, deleted, err := s.Storage.Get(req.Context(), linkHash)
	// 400.
	if err != nil {
		logger.PrintLog(logger.WARN, "Get exception: "+err.Error(), s.LogEnabled)
		httpResp.BadRequest(res)
		return
	}

	// 410.
	if deleted {
		logger.PrintLog(logger.INFO, "Current item was deleted", s.LogEnabled)
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
		logger.PrintLog(logger.INFO, "Success", s.LogEnabled)
		httpResp.TempRedirect(res, additional)
		return
	}

	// Если нет, отдаем BadRequest 400.
	logger.PrintLog(logger.WARN, "Not success", s.LogEnabled)
	httpResp.BadRequest(res)
}

// HandlePOST - принимает запрос, наполняет объект данными, выполняет запрос к хранилищу.
// Эндпоинт с методом POST и путём /.
// Сервер принимает в теле запроса строку URL как text/plain
// и возвращает ответ с кодом 201 и сокращённым URL как text/plain.
func (s *Server) HandlePOST(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandlePOST`, s.LogEnabled)

	if s.ShutdownProcess {
		httpResp.Shutdown(res)
		return
	}

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	if len(contentBody) == 0 {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkHash := sha1hash.Create(string(contentBody), 8)
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkHash)
	originalLink := string(contentBody)
	userID := req.Context().Value(cookie.UserNum(`UserID`))

	if userID == nil {
		userID = 0
	}

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: shortLink,
	}

	setErr := s.Storage.Set(req.Context(), originalLink, shortLink, linkHash, userID.(int))

	if setErr != nil {
		var pgErrType *pgconn.PgError
		if errors.As(setErr, &pgErrType) {
			if pgErrType.Code == pgerrcode.UniqueViolation {
				txt := fmt.Sprintf(`Can not set link data: %s, body: %s, linkID: %s`, setErr.Error(), string(contentBody), linkHash)
				logger.PrintLog(logger.WARN, txt, s.LogEnabled)
				httpResp.Conflict(res, additional)
				return
			}
		}
		logger.PrintLog(logger.ERROR, "Can not set link data: "+setErr.Error(), s.LogEnabled)
		httpResp.BadRequest(res)
		return
	}

	err := req.Body.Close()
	if err != nil {
		logger.PrintLog(logger.ERROR, "Body close: "+err.Error(), s.LogEnabled)
		httpResp.BadRequest(res)
		return
	}

	// Отдаем 201 ответ с шортлинком.
	httpResp.Created(res, additional)
}

// HandleAPIUserUrlsDelete - функция метода HandleAPI, агрегирующая логику удаления пакета URL через API.
// Работает для конкретного пользователя.
// хендлер DELETE /api/user/urls в теле запроса принимает список идентификаторов сокращённых URL
// для асинхронного удаления. Запрос может быть таким:
// DELETE http://localhost:8080/api/user/urls
// Content-Type: application/json
//
// ["6qxTVvsy", "RTfd56hn", "Jlfd67ds"]
func (s *Server) HandleAPIUserUrlsDelete(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandleAPIUserUrlsDelete`, s.LogEnabled)

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	httpResp.Accepted(res, httpResp.Additional{})

	err := req.Body.Close()
	if err != nil {
		httpResp.BadRequest(res)
		return
	}

	userID := req.Context().Value(cookie.UserNum(`UserID`))
	s.Storage.HandleUserUrlsDelete(string(contentBody), userID.(int))
}

// HandleAPIUserUrls - функция метода HandleAPI, агрегирующая логику обработки пакета URL через API.
// Работает для конкретного пользователя.
// Хендлер GET /api/user/urls может вернуть пользователю все когда-либо сокращённые им URL в формате:
// [
//
//	{
//	    "short_url": "http://...",
//	    "original_url": "http://..."
//	},
//	...
//
// ]
func (s *Server) HandleAPIUserUrls(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandleAPIUserUrls`, s.LogEnabled)

	userID := req.Context().Value(cookie.UserNum(`UserID`))
	byteRes, err := s.Storage.HandleUserUrls(req.Context(), userID.(int))

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
// хендлер POST /api/shorten/batch, принимает в теле запроса множество URL для сокращения в формате:
// [
//
//	{
//	    "correlation_id": "<строковый идентификатор>",
//	    "original_url": "<URL для сокращения>"
//	},
//	...
//
// ]
// В качестве ответа хендлер возвращает данные в формате:
// [
//
//	{
//	    "correlation_id": "<строковый идентификатор из объекта запроса>",
//	    "short_url": "<результирующий сокращённый URL>"
//	},
//	...
//
// ]
func (s *Server) HandleAPIBatch(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandleAPIBatch`, s.LogEnabled)

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			httpResp.BadRequest(res)
			return
		}
	}(req.Body)

	logger.PrintLog(logger.DEBUG, `Body: `+string(contentBody), s.LogEnabled)

	userID := req.Context().Value(cookie.UserNum(`UserID`))

	resData, err := s.Storage.BatchSet(req.Context(), contentBody, userID.(int))

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
// Эндпоинт POST /api/shorten будет принимать в теле запроса JSON-объект {"url":"<some_url>"}
// и возвращать в ответ объект {"result":"<short_url>"}.
// Content-Type: application/json
func (s *Server) HandleAPIShorten(res http.ResponseWriter, req *http.Request) {

	logger.PrintLog(logger.DEBUG, `HandleAPIShorten`, s.LogEnabled)

	contentBody, errBody := io.ReadAll(req.Body)
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
	linkHash := sha1hash.Create(apiData.URL, 8)
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkHash)
	originalLink := apiData.URL
	userID := req.Context().Value(cookie.UserNum(`UserID`))

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

	err = s.Storage.Set(req.Context(), originalLink, shortLink, linkHash, userID.(int))

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == `23505` {
				logger.PrintLog(logger.WARN, "Can not set link data: "+err.Error(), s.LogEnabled)
				httpResp.ConflictJSON(res, additional)
				return
			}
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error(), s.LogEnabled)
			httpResp.BadRequest(res)
			return
		}
	}

	err = req.Body.Close()
	if err != nil {
		httpResp.BadRequest(res)
		return
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

	ok, err := s.Storage.Ping(req.Context())
	if ok {
		httpResp.Ok(res)
		return
	}
	if err != nil {
		logger.PrintLog(logger.ERROR, handlePingErr.Error()+`: `+err.Error(), s.LogEnabled)
	}
}

// HandleOther - middleware для обработки неожидаемых запросов.
func HandleOther(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		logger.PrintLog(logger.DEBUG, `Method: `+req.Method, true)
		//body, _ := io.ReadAll(req.Body)
		//defer req.Body.Close()
		//logger.PrintLog(logger.DEBUG, `Body: `+string(body), true)
		//logger.PrintLog(logger.DEBUG, req.Method, true)
		if req.Method == http.MethodGet || req.Method == http.MethodPost || req.Method == http.MethodDelete {
			next.ServeHTTP(res, req)
			return
		}
		httpResp.BadRequest(res)
	})
}

// ChooseStorage - метод выбора хранилища в зависимости от параметров конфигурации.
// Работает с конфигурацией, которую принимает вторым параметром.
// Возвращает модель для работы с хранилищем и ошибку.
func ChooseStorage(ctx context.Context, conf confModule.OuterConfig) (model.Storable, error) {
	pgCsErr := &ErrorHandlers{
		layer:          `Handlers`,
		funcName:       `ChooseStorage`,
		parentFuncName: `-`,
	}

	var storage model.Storable

	if conf.Env.DB != "" || conf.Flag.DB != "" {
		pgPool, err := db.Connect(ctx, conf)
		if err != nil {
			return nil, err
		}
		storage = &database.DBStorage{
			ConnectionPool: pgPool,
			Cfg:            conf,
		}
		err = storage.Init()
		if err != nil {
			pgCsErr.message = `can't init DB storage`
			return storage, fmt.Errorf(pgCsErr.Error()+`: %w`, err)
		}

		go storage.AsyncSaver()
		return storage, nil
	}

	if conf.Env.LinkFile != `` || conf.Flag.LinkFile != `` {
		storage = &files.FileStorage{
			Cfg: conf,
		}
		err := storage.Init()
		if err != nil {
			pgCsErr.message = `can't init file storage`
			return storage, fmt.Errorf(pgCsErr.Error()+`: %w`, err)
		}
		return storage, nil
	}

	storage = &memory.MemStorage{
		Storage: memoryStorage.Storage{},
		Cfg:     conf,
	}
	err := storage.Init()
	if err != nil {
		pgCsErr.message = `can't init memory storage`
		return storage, fmt.Errorf(pgCsErr.Error()+`: %w`, err)
	}
	return storage, nil
}

// Server - основная структура сервера, определяющая его работу.
type Server struct {
	Storage         model.Storable
	Routers         chi.Router
	Config          confModule.OuterConfig
	HTTP            http.Server
	LogEnabled      bool
	ShutdownProcess bool
}

// Init - инициализирует сервер параметрами.
func (s *Server) Init(cfg confModule.OuterConfig, needLogging bool) error {
	handleInitErr := &ErrorHandlers{
		layer:          `Handlers`,
		funcName:       `Init`,
		parentFuncName: `-`,
	}

	var emptyCfg = confModule.OuterConfig{}
	if cfg == emptyCfg {
		return handleInitErr
	}
	s.Config = cfg
	s.LogEnabled = needLogging
	s.ShutdownProcess = false
	return nil
}

// Start - запускает сервер в работу.
func (s *Server) Start(ctx context.Context) error {
	if s.ShutdownProcess {
		return nil
	}

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
		With(compress.GzipHandler).
		With(HandleOther)
	if s.LogEnabled {
		s.Routers.With(extlogger.Log)
	}
	s.Routers.Route("/", func(r chi.Router) {
		//s.Routers.Group(func(r chi.Router) {
		//	r.HandleFunc("/debug/pprof/*", pprof.Index)
		//	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		//	r.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		//})
		s.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthSetter)

			r.Post(`/`, s.HandlePOST)
			r.Post(`/api/{query}`, s.HandleAPIShorten)
			r.Post(`/api/shorten/{query}`, s.HandleAPIBatch)
			r.Get(`/ping`, s.HandlePing)
			r.Get(`/{query}`, s.HandleGET)
		})
		s.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthChecker)

			r.Delete(`/api/user/{query}`, s.HandleAPIUserUrlsDelete)
			r.Get(`/api/user/{query}`, s.HandleAPIUserUrls)
		})
	})

	s.HTTP.Addr = s.Config.Final.AppAddr
	s.HTTP.Handler = s.Routers
	err = s.HTTP.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) && s.ShutdownProcess {
			return nil
		}
		return fmt.Errorf(`can't start http listener: %w`, err)
	}

	return nil
}

// Stop - останавливает сервер.
func (s *Server) Stop(ctx context.Context) error {
	s.ShutdownProcess = true
	s.Storage.Destroy()
	err := s.HTTP.Shutdown(ctx)
	if err != nil {
		return err
	}
	return nil
}
