package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/MaximMNsk/go-url-shortener/internal/models/interface/models/mocks"
	pb "github.com/MaximMNsk/go-url-shortener/proto"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
)

func TestShortenerServer_Ping(t *testing.T) {
	type args struct {
		pingResult bool
		pingError  error
	}
	type want struct {
		resp pb.PingResponse
		err  error
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "ping successfully",
			args: args{
				pingResult: true,
				pingError:  nil,
			},
			want: want{
				resp: pb.PingResponse{
					Result: pb.Result_HTTP_200_OK,
				},
				err: nil,
			},
		},
		{
			name: "ping wrong",
			args: args{
				pingResult: false,
				pingError:  nil,
			},
			want: want{
				resp: pb.PingResponse{
					Result: pb.Result_HTTP_503_UNAVAILABLE,
				},
				err: nil,
			},
		},
		{
			name: "ping error",
			args: args{
				pingResult: false,
				pingError:  errors.New(`some error`),
			},
			want: want{
				resp: pb.PingResponse{
					Result: pb.Result_HTTP_500_INTERNAL_ERROR,
				},
				err: errors.New(`some error`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			ctx := context.Background()
			err := serv.Init(ctx)
			require.NoError(t, err)

			storageMock := mocks.NewStorable(t)
			storageMock.
				On(`Ping`, mock.Anything).
				Return(tt.args.pingResult, tt.args.pingError)
			serv.Storage = storageMock
			ping, err := serv.Ping(ctx, &pb.PingRequest{})
			if tt.name == `ping error` {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, ping.Result, tt.want.resp.Result)
		})
	}
}

func TestShortenerServer_Stat(t *testing.T) {
	type args struct {
		ctx        context.Context
		mockNeed   bool
		mockError  error
		mockResult []byte
	}
	type want struct {
		resp pb.StatResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Ok",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": `127.0.0.1`})),
				mockNeed:   true,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":10}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_200_OK,
				},
			},
		},
		{
			name: "Empty metadata",
			args: args{
				ctx:        context.Background(),
				mockNeed:   false,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":10}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_500_INTERNAL_ERROR,
				},
			},
		},
		{
			name: "Wrong net",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": `192.0.0.1`})),
				mockNeed:   false,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":10}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_403_FORBIDDEN,
				},
			},
		},
		{
			name: "Empty IP",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": ``})),
				mockNeed:   false,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":10}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_403_FORBIDDEN,
				},
			},
		},
		{
			name: "Wrong subnet",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": `127.0.0.1`})),
				mockNeed:   false,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":10}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_500_INTERNAL_ERROR,
				},
			},
		},
		{
			name: "Wrong data source",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": `127.0.0.1`})),
				mockNeed:   true,
				mockError:  errors.New(`some error`),
				mockResult: nil,
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Wrong source response",
			args: args{
				ctx:        metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Real-IP": `127.0.0.1`})),
				mockNeed:   true,
				mockError:  nil,
				mockResult: []byte("{\"urls\":100,\"users\":}"),
			},
			want: want{
				resp: pb.StatResponse{
					Stats: &pb.Stats{
						Users: 10,
						Urls:  100,
					},
					Result: pb.Result_HTTP_500_INTERNAL_ERROR,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.name == `Wrong subnet` {
				serv.Config.Final.TrustedSubnet = `localhost`
			}

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`HandleStats`, mock.Anything).
					Return(tt.args.mockResult, tt.args.mockError)
				serv.Storage = storageMock
			}

			ping, err := serv.Stat(tt.args.ctx, &pb.StatRequest{})

			require.NoError(t, err)
			assert.Equal(t, tt.want.resp.Result, ping.Result)
		})
	}
}

func TestShortenerServer_SetShort(t *testing.T) {
	type args struct {
		ctx           context.Context
		url           string
		mockNeed      bool
		mockError     error
		mockResult    []byte
		willErrReturn bool
	}
	type want struct {
		resp pb.SetShortResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Empty url",
			args: args{
				ctx:      context.Background(),
				mockNeed: false,
				url:      ``,
			},
			want: want{
				resp: pb.SetShortResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
					Data:   ``,
				},
			},
		},
		{
			name: "Resource error",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				url:           `qwerty`,
				mockNeed:      true,
				mockError:     errors.New(`unexpected error`),
				willErrReturn: true,
			},
			want: want{
				resp: pb.SetShortResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
					Data:   ``,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				url:           `asdasdasd`,
				mockNeed:      true,
				mockError:     nil,
				willErrReturn: false,
			},
			want: want{
				resp: pb.SetShortResponse{
					Result: pb.Result_HTTP_201_CREATED,
					Data:   `http://localhost:8080/00ea1da4`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`Set`, tt.args.ctx, tt.args.url, mock.Anything, mock.Anything, mock.Anything).
					Return(tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.SetShort(tt.args.ctx, &pb.SetShortRequest{
				OriginalURL: tt.args.url,
			})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
			assert.Equal(t, tt.want.resp.Data, resp.Data)
		})
	}
}

func TestShortenerServer_GetShort(t *testing.T) {
	type args struct {
		ctx           context.Context
		short         string
		mockNeed      bool
		mockShort     string
		mockError     error
		mockResult    string
		mockIsDeleted bool
		willErrReturn bool
	}
	type want struct {
		resp pb.GetShortResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Incorrect url",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				short:         `asd*asd\qwe@asd^321`,
				mockNeed:      false,
				willErrReturn: true,
			},
			want: want{
				resp: pb.GetShortResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Empty url",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				short:         ``,
				mockNeed:      false,
				willErrReturn: true,
			},
			want: want{
				resp: pb.GetShortResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Resource error",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), `-1`),
				short:         `http://localhost:8080/00ea1da4`,
				mockShort:     `00ea1da4`,
				mockNeed:      true,
				mockError:     errors.New(`unexpected error`),
				mockResult:    mock.Anything,
				mockIsDeleted: false,
				willErrReturn: true,
			},
			want: want{
				resp: pb.GetShortResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Deleted url",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), `-1`),
				short:         `http://localhost:8080/00ea1da4`,
				mockShort:     `00ea1da4`,
				mockNeed:      true,
				mockError:     nil,
				mockResult:    mock.Anything,
				mockIsDeleted: true,
				willErrReturn: false,
			},
			want: want{
				resp: pb.GetShortResponse{
					Result: pb.Result_HTTP_410_GONE,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), `-1`),
				short:         `http://localhost:8080/00ea1da4`,
				mockShort:     `00ea1da4`,
				mockNeed:      true,
				mockError:     nil,
				mockResult:    `asdasdasd`,
				mockIsDeleted: false,
				willErrReturn: false,
			},
			want: want{
				resp: pb.GetShortResponse{
					Result: pb.Result_HTTP_307_TMP_REDIRECT,
					Data:   `asdasdasd`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`Get`, tt.args.ctx, tt.args.mockShort).
					Return(tt.args.mockResult, tt.args.mockIsDeleted, tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.GetShort(tt.args.ctx, &pb.GetShortRequest{
				ShortURL: tt.args.short,
			})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
			assert.Equal(t, tt.want.resp.Data, resp.Data)
		})
	}
}

func TestShortenerServer_APIBatch(t *testing.T) {
	type args struct {
		ctx           context.Context
		input         []*pb.BeforeShort
		output        []*pb.AfterShort
		mockNeed      bool
		mockShort     []byte
		mockError     error
		mockResult    []byte
		mockIsDeleted bool
		willErrReturn bool
	}
	type want struct {
		resp pb.APIBatchResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Resource error",
			args: args{
				ctx: context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				input: []*pb.BeforeShort{
					{
						CorrelationId: `00ea1da4`,
						OriginalUrl:   `asdasdasd`,
					},
				},
				mockNeed:      true,
				mockShort:     []byte(`[{"correlation_id":"00ea1da4","original_url":"asdasdasd"}]`),
				mockError:     errors.New(`unexpected error`),
				willErrReturn: true,
			},
			want: want{
				resp: pb.APIBatchResponse{
					Result:    pb.Result_HTTP_400_BAD_REQUEST,
					ShortURLs: nil,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx: context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				input: []*pb.BeforeShort{
					{
						CorrelationId: `00ea1da4`,
						OriginalUrl:   `asdasdasd`,
					},
				},
				mockNeed:      true,
				mockShort:     []byte(`[{"correlation_id":"00ea1da4","original_url":"asdasdasd"}]`),
				mockError:     nil,
				mockResult:    []byte(`[{"correlation_id":"00ea1da4","short_url":"http://localhost:8080/00ea1da4"}]`),
				willErrReturn: false,
			},
			want: want{
				resp: pb.APIBatchResponse{
					Result: pb.Result_HTTP_200_OK,
					ShortURLs: []*pb.AfterShort{
						{
							CorrelationId: `00ea1da4`,
							ShortUrl:      `http://localhost:8080/00ea1da4`,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`BatchSet`, tt.args.ctx, tt.args.mockShort, -1).
					Return(tt.args.mockResult, tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.APIBatch(tt.args.ctx, &pb.APIBatchRequest{
				URLs: tt.args.input,
			})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
			assert.Equal(t, tt.want.resp.ShortURLs, resp.ShortURLs)
		})
	}
}

func TestShortenerServer_APIShorten(t *testing.T) {
	type args struct {
		ctx           context.Context
		input         []*pb.InputURL
		output        []*pb.OutputURL
		mockNeed      bool
		mockURL       string
		mockError     error
		mockResult    []byte
		mockIsDeleted bool
		willErrReturn bool
	}
	type want struct {
		resp pb.APIShortenResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Resource error",
			args: args{
				ctx: context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				input: []*pb.InputURL{
					{
						Url: `asdasdasd`,
					},
				},
				output: []*pb.OutputURL{
					{
						Result: `http://localhost:8080/00ea1da4`,
					},
				},
				mockNeed:      true,
				mockError:     errors.New(`unexpected error`),
				willErrReturn: true,
			},
			want: want{
				resp: pb.APIShortenResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx: context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				input: []*pb.InputURL{
					{
						Url: `asdasdasd`,
					},
				},
				output: []*pb.OutputURL{
					{
						Result: `http://localhost:8080/da39a3ee`,
					},
				},
				mockNeed:      true,
				mockError:     nil,
				willErrReturn: false,
			},
			want: want{
				resp: pb.APIShortenResponse{
					Result: pb.Result_HTTP_200_OK,
					Shorten: &pb.OutputURL{
						Result: `http://localhost:8080/da39a3ee`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`Set`, tt.args.ctx, tt.args.mockURL, mock.Anything, mock.Anything, mock.Anything).
					Return(tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.APIShorten(tt.args.ctx, &pb.APIShortenRequest{
				URL: &pb.InputURL{
					Url: tt.args.mockURL,
				},
			})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
			assert.Equal(t, tt.want.resp.Shorten, resp.Shorten)
		})
	}
}

func TestShortenerServer_APIUserUrls(t *testing.T) {
	type args struct {
		ctx           context.Context
		mockNeed      bool
		mockError     error
		mockResult    []byte
		willErrReturn bool
	}
	type want struct {
		resp pb.APIUserUrlsResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Resource error",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				mockNeed:      true,
				mockError:     errors.New(`unexpected error`),
				mockResult:    nil,
				willErrReturn: true,
			},
			want: want{
				resp: pb.APIUserUrlsResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
					Data:   nil,
				},
			},
		},
		{
			name: "No content",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				mockNeed:      true,
				mockError:     nil,
				mockResult:    nil,
				willErrReturn: false,
			},
			want: want{
				resp: pb.APIUserUrlsResponse{
					Result: pb.Result_HTTP_204_NO_CONTENT,
					Data:   nil,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx:           context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				mockNeed:      true,
				mockError:     nil,
				mockResult:    []byte(`[{"short_url":"http://localhost:8080/00ea1da4","original_url":"asdasdasd"}]`),
				willErrReturn: false,
			},
			want: want{
				resp: pb.APIUserUrlsResponse{
					Result: pb.Result_HTTP_200_OK,
					Data: []*pb.UserURL{
						{
							ShortUrl:    `http://localhost:8080/00ea1da4`,
							OriginalUrl: `asdasdasd`,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`HandleUserUrls`, tt.args.ctx, -1).
					Return(tt.args.mockResult, tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.APIUserUrls(tt.args.ctx, &pb.APIUserUrlsRequest{})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
			assert.Equal(t, tt.want.resp.Data, resp.Data)
		})
	}
}

func TestShortenerServer_APIUserURLsDelete(t *testing.T) {
	type args struct {
		ctx           context.Context
		mockNeed      bool
		mockResult    any
		mockError     any
		willErrReturn bool
		URLs          []string
	}
	type want struct {
		resp pb.APIUserURLsDeleteResponse
	}

	tests := []*struct {
		name string
		args args
		want want
	}{
		{
			name: "Empty request",
			args: args{
				ctx:      context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				mockNeed: false,
				URLs:     []string{},
			},
			want: want{
				resp: pb.APIUserURLsDeleteResponse{
					Result: pb.Result_HTTP_400_BAD_REQUEST,
				},
			},
		},
		{
			name: "Ok",
			args: args{
				ctx:      context.WithValue(context.Background(), cookie.UserNum(`UserID`), -1),
				mockNeed: false,
				URLs: []string{
					`asd`, `zxc`,
				},
			},
			want: want{
				resp: pb.APIUserURLsDeleteResponse{
					Result: pb.Result_HTTP_200_OK,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serv ShortenerServer
			err := serv.Init(tt.args.ctx)
			require.NoError(t, err)

			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.
					On(`HandleUserUrls`, tt.args.ctx, -1).
					Return(tt.args.mockResult, tt.args.mockError)
				serv.Storage = storageMock
			}

			resp, err := serv.APIUserURLsDelete(tt.args.ctx, &pb.APIUserURLsDeleteRequest{
				URL: tt.args.URLs,
			})

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.resp.Result, resp.Result)
		})
	}
}

var Serv ShortenerServer

func TestShortenerServer_Start(t *testing.T) {
	type args struct {
		willErrReturn bool
		appAddr       string
	}
	type want struct {
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Start error",
			args: args{
				willErrReturn: true,
				appAddr:       "lihgo;irg",
			},
			want: want{},
		},
		//{
		//	name: "Start ok",
		//	args: args{
		//		willErrReturn: false,
		//		appAddr:       "localhost:8080",
		//	},
		//	want: want{},
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Serv.Init(context.Background())
			require.NoError(t, err)

			Serv.Config.Final.AppAddr = tt.args.appAddr

			err = Serv.Start()

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShortenerServer_Stop(t *testing.T) {
	type args struct {
		willErrReturn bool
		mockNeed      bool
		mockError     error
	}
	type want struct {
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Stopping err",
			args: args{
				willErrReturn: true,
				mockError:     errors.New(`unexpected error`),
				mockNeed:      true,
			},
			want: want{},
		},
		//{
		//	name: "Stopping ok",
		//	args: args{
		//		willErrReturn: false,
		//	},
		//	want: want{},
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockNeed {
				storageMock := mocks.NewStorable(t)
				storageMock.On(`Destroy`).Return(tt.args.mockError)
				Serv.Storage = storageMock
			}
			err := Serv.Stop()

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
