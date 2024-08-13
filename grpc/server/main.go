package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/models/interface/models"
	"github.com/MaximMNsk/go-url-shortener/internal/models/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	memoryStorage "github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	pb "github.com/MaximMNsk/go-url-shortener/proto"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/auth/mtd"
	cfg "github.com/MaximMNsk/go-url-shortener/server/config"
)

type ShortenerServer struct {
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedShortenerServer
	Storage         models.Storable
	Config          cfg.OuterConfig
	GRPC            *grpc.Server
	LogEnabled      bool
	ShutdownProcess bool
	Mu              sync.RWMutex
}

// GetShort - метод получения URL из ShortURL.
func (s *ShortenerServer) GetShort(ctx context.Context, in *pb.GetShortRequest) (*pb.GetShortResponse, error) {
	if s.ShutdownProcess {
		return &pb.GetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	inputURL, err := url.Parse(in.ShortURL)
	if err != nil {
		return &pb.GetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}
	if inputURL.Path == `` {
		return &pb.GetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, errors.New("invalid short url")
	}

	saved, deleted, err := s.Storage.Get(ctx, inputURL.Path[1:])
	if err != nil {
		return &pb.GetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	if deleted {
		return &pb.GetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_410_GONE,
		}, nil
	}

	return &pb.GetShortResponse{
		Data:   saved,
		Result: pb.Result_HTTP_307_TMP_REDIRECT,
	}, nil
}

// SetShort - принимает данные клиента и выполняет запрос к хранилищу.
func (s *ShortenerServer) SetShort(ctx context.Context, in *pb.SetShortRequest) (*pb.SetShortResponse, error) {
	if s.ShutdownProcess {
		return &pb.SetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	if len(in.OriginalURL) == 0 {
		return &pb.SetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, nil
	}

	linkHash := sha1hash.Create(in.OriginalURL, 8)
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkHash)
	userID := ctx.Value(cookie.UserNum(`UserID`)).(int)

	err := s.Storage.Set(ctx, in.OriginalURL, shortLink, linkHash, userID)
	if err != nil {
		var pgErrType *pgconn.PgError
		if errors.As(err, &pgErrType) {
			if pgErrType.Code == pgerrcode.UniqueViolation {
				return &pb.SetShortResponse{
					Data:   ``,
					Result: pb.Result_HTTP_409_CONFLICT,
				}, nil
			}
		}
		return &pb.SetShortResponse{
			Data:   ``,
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	return &pb.SetShortResponse{
		Data:   shortLink,
		Result: pb.Result_HTTP_201_CREATED,
	}, nil
}

// APIUserURLsDelete - агрегирует логику удаления пакета URL через API.
// Запрос - ["6qxTVvsy", "RTfd56hn", "Jlfd67ds"]
func (s *ShortenerServer) APIUserURLsDelete(ctx context.Context, in *pb.APIUserURLsDeleteRequest) (*pb.APIUserURLsDeleteResponse, error) {
	if s.ShutdownProcess {
		return &pb.APIUserURLsDeleteResponse{
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	content := in.URL

	if len(content) == 0 {
		return &pb.APIUserURLsDeleteResponse{
			Result: pb.Result_HTTP_400_BAD_REQUEST,
		}, nil
	}

	contentString := strings.Join(content, ",")

	userID := ctx.Value(cookie.UserNum(`UserID`)).(int)
	s.Storage.HandleUserUrlsDelete(contentString, userID)

	return &pb.APIUserURLsDeleteResponse{
		Result: pb.Result_HTTP_200_OK,
	}, nil
}

// APIUserUrls - агрегирует логику обработки пакета URL через API.
// Ответ - [{"short_url": "http://...","original_url": "http://..."}]
func (s *ShortenerServer) APIUserUrls(ctx context.Context, _ *pb.APIUserUrlsRequest) (*pb.APIUserUrlsResponse, error) {
	if s.ShutdownProcess {
		return &pb.APIUserUrlsResponse{
			Data:   nil,
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	userID := ctx.Value(cookie.UserNum(`UserID`)).(int)
	byteRes, err := s.Storage.HandleUserUrls(ctx, userID)
	if err != nil {
		return &pb.APIUserUrlsResponse{
			Result: pb.Result_HTTP_400_BAD_REQUEST,
			Data:   nil,
		}, err
	}

	if byteRes == nil {
		return &pb.APIUserUrlsResponse{
			Result: pb.Result_HTTP_204_NO_CONTENT,
			Data:   nil,
		}, err
	}

	var resultData []*pb.UserURL

	err = json.Unmarshal(byteRes, &resultData)
	if err != nil {
		return &pb.APIUserUrlsResponse{
			Result: pb.Result_HTTP_500_INTERNAL_ERROR,
			Data:   nil,
		}, err
	}

	return &pb.APIUserUrlsResponse{
		Result: pb.Result_HTTP_200_OK,
		Data:   resultData,
	}, err
}

// APIBatch - агрегирует логику обработки пакета URL через API.
// Запрос - [{"correlation_id": "<строковый идентификатор>","original_url": "<URL для сокращения>"}]
// Ответ - [{"correlation_id": "<строковый идентификатор из объекта запроса>","short_url": "<результирующий сокращённый URL>"}]
func (s *ShortenerServer) APIBatch(ctx context.Context, in *pb.APIBatchRequest) (*pb.APIBatchResponse, error) {
	if s.ShutdownProcess {
		return &pb.APIBatchResponse{
			ShortURLs: nil,
			Result:    pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	userID := ctx.Value(cookie.UserNum(`UserID`)).(int)
	contentBody, err := json.Marshal(in.URLs)
	if err != nil {
		return &pb.APIBatchResponse{
			ShortURLs: nil,
			Result:    pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	resData, err := s.Storage.BatchSet(ctx, contentBody, userID)
	if err != nil {
		return &pb.APIBatchResponse{
			ShortURLs: nil,
			Result:    pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	var resultData []*pb.AfterShort
	err = json.Unmarshal(resData, &resultData)
	if err != nil {
		return &pb.APIBatchResponse{
			ShortURLs: nil,
			Result:    pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	return &pb.APIBatchResponse{
		ShortURLs: resultData,
		Result:    pb.Result_HTTP_200_OK,
	}, nil
}

// APIShorten - агрегирует логику обработки одного URL через API.
// Запрос - {"url":"<some_url>"}.
// Ответ - {"result":"<short_url>"}.
func (s *ShortenerServer) APIShorten(ctx context.Context, in *pb.APIShortenRequest) (*pb.APIShortenResponse, error) {
	if s.ShutdownProcess {
		return &pb.APIShortenResponse{
			Shorten: nil,
			Result:  pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	linkHash := sha1hash.Create(in.URL.Url, 8)
	shortLink := shorter.GetShortURL(s.Config.Final.ShortURLAddr, linkHash)
	originalLink := in.URL.Url
	userID := ctx.Value(cookie.UserNum(`UserID`)).(int)

	err := s.Storage.Set(ctx, originalLink, shortLink, linkHash, userID)
	if err != nil {
		return &pb.APIShortenResponse{
			Shorten: nil,
			Result:  pb.Result_HTTP_400_BAD_REQUEST,
		}, err
	}

	return &pb.APIShortenResponse{
		Shorten: &pb.OutputURL{Result: shortLink},
		Result:  pb.Result_HTTP_200_OK,
	}, nil
}

// Ping - метод сервера для проверки доступности хранилища.
func (s *ShortenerServer) Ping(ctx context.Context, _ *pb.PingRequest) (*pb.PingResponse, error) {
	if s.ShutdownProcess {
		return &pb.PingResponse{
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	ok, err := s.Storage.Ping(ctx)
	if err != nil {
		return &pb.PingResponse{
			Result: pb.Result_HTTP_500_INTERNAL_ERROR,
		}, err
	}
	if !ok {
		return &pb.PingResponse{
			Result: pb.Result_HTTP_503_UNAVAILABLE,
		}, nil
	}

	return &pb.PingResponse{
		Result: pb.Result_HTTP_200_OK,
	}, nil
}

// Stat - метод отображения статистики.
// Читает метаданные из ключа X-Real-IP.
// Ответ - {"urls":<int>, "users":<int>}
func (s *ShortenerServer) Stat(ctx context.Context, _ *pb.StatRequest) (*pb.StatResponse, error) {
	if s.ShutdownProcess {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_503_UNAVAILABLE,
			Stats:  nil,
		}, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_500_INTERNAL_ERROR,
			Stats:  nil,
		}, nil
	}

	headerIP := md.Get("X-Real-IP")
	if len(headerIP) == 0 {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_403_FORBIDDEN,
			Stats:  nil,
		}, nil
	}
	realIP := net.ParseIP(headerIP[0])
	_, iPPool, err := net.ParseCIDR(s.Config.Final.TrustedSubnet)
	if err != nil {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_500_INTERNAL_ERROR,
			Stats:  nil,
		}, nil
	}
	if !iPPool.Contains(realIP) {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_403_FORBIDDEN,
			Stats:  nil,
		}, nil
	}

	stats, err := s.Storage.HandleStats(ctx)
	if err != nil {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_400_BAD_REQUEST,
			Stats:  nil,
		}, nil
	}
	var unmarshalledStats *pb.Stats
	err = json.Unmarshal(stats, &unmarshalledStats)
	if err != nil {
		return &pb.StatResponse{
			Result: pb.Result_HTTP_500_INTERNAL_ERROR,
			Stats:  nil,
		}, nil
	}

	return &pb.StatResponse{
		Result: pb.Result_HTTP_200_OK,
		Stats:  unmarshalledStats,
	}, nil
}

// ChooseStorage - метод выбора хранилища в зависимости от параметров конфигурации.
// Работает с конфигурацией, которую принимает вторым параметром.
// Возвращает ошибку.
func (s *ShortenerServer) ChooseStorage(ctx context.Context, conf cfg.OuterConfig) error {
	if conf.Env.DB != "" || conf.Flag.DB != "" || conf.ConfFile.DB != "" {
		logger.PrintLog(logger.INFO, `DB storage`, s.LogEnabled)
		pgPool, err := db.NewPool(ctx, conf)
		if err != nil {
			return err
		}
		s.Storage = &database.DBStorage{
			ConnectionPool: pgPool,
			Cfg:            conf,
		}
		err = s.Storage.Init()
		if err != nil {
			return err
		}

		go s.Storage.AsyncSaver()
		return nil
	}

	if conf.Env.LinkFile != `` || conf.Flag.LinkFile != `` || conf.ConfFile.LinkFile != `` {
		logger.PrintLog(logger.INFO, `File storage`, s.LogEnabled)
		s.Storage = &files.FileStorage{
			Cfg: conf,
		}
		err := s.Storage.Init()
		if err != nil {
			return err
		}
		return nil
	}
	logger.PrintLog(logger.INFO, `InMem storage`, s.LogEnabled)
	s.Storage = &memory.MemStorage{
		Storage: memoryStorage.Storage{},
		Cfg:     conf,
	}
	err := s.Storage.Init()
	if err != nil {
		return err
	}
	return nil
}

// Init - инициализирует сервер параметрами.
func (s *ShortenerServer) Init(ctx context.Context) error {
	s.ShutdownProcess = false
	s.LogEnabled = true
	err := s.Config.InitConfig(false)
	if err != nil {
		return err
	}
	err = s.ChooseStorage(ctx, s.Config)
	if err != nil {
		return err
	}
	s.GRPC = grpc.NewServer(
		grpc.UnaryInterceptor(mtd.JWTInterceptor),
	)
	return nil
}

// Start - запускает сервер в работу.
func (s *ShortenerServer) Start() error {
	listener, err := net.Listen("tcp", s.Config.Final.AppAddr)
	if err != nil {
		return err
	}
	pb.RegisterShortenerServer(s.GRPC, s)
	if err := s.GRPC.Serve(listener); err != nil {
		return err
	}
	return nil
}

// Stop - останавливает сервер.
func (s *ShortenerServer) Stop() error {
	err := s.Storage.Destroy()
	if err != nil {
		return err
	}
	s.GRPC.Stop()
	return nil
}

func main() {
	ctx := context.Background()
	serv := new(ShortenerServer)
	err := serv.Init(ctx)
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error(), serv.LogEnabled)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		for {
			select {
			case <-exit:
				logger.PrintLog(logger.INFO, "stopping server", serv.LogEnabled)
				stopped := serv.Stop()
				if stopped != nil {
					logger.PrintLog(logger.INFO, "incorrectly stopping server", serv.LogEnabled)
				}
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
	}()

	logger.PrintLog(logger.INFO, `Start server`, serv.LogEnabled)
	err = serv.Start()
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error(), true)
	}
}
