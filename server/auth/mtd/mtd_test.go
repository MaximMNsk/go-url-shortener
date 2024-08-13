package mtd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestGetUserID(t *testing.T) {
	type args struct {
		userID int
	}
	type want struct {
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Ok",
			args: args{
				userID: 200,
			},
			want: want{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := BuildJWTString(tt.args.userID)
			require.NoError(t, err)
			UID := GetUserID(token)
			require.Equal(t, UID, tt.args.userID)
		})
	}
}

func TestJWTInterceptor(t *testing.T) {
	type args struct {
		willErrReturn bool
		authCred      string
	}
	type want struct {
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Bad",
			args: args{
				willErrReturn: true,
				authCred:      "Bearer bad",
			},
			want: want{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := BuildJWTString(100)
			require.NoError(t, err)
			var authCred string
			if len(tt.args.authCred) == 0 {
				authCred = `Bearer ` + token
			} else {
				authCred = tt.args.authCred
			}
			ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(map[string]string{
				`authorization`: authCred,
			}))

			var h grpc.UnaryHandler
			interceptor, err := JWTInterceptor(
				ctx,
				nil,
				nil,
				h,
			)

			t.Log(interceptor)

			if tt.args.willErrReturn {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
