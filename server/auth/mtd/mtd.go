package mtd

import (
	"context"
	"errors"
	"strings"

	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID int) (string, error) {
	return cookie.BuildJWTString(userID)
}

// GetUserID - получает UserID из токена.
func GetUserID(tokenString string) int {
	return cookie.GetUserID(tokenString)
}

func JWTInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New(`broken context metadata`)
	}

	reqToken := ``
	bearerTokens := md[`authorization`]
	if len(bearerTokens) > 0 {
		splitToken := strings.Split(bearerTokens[0], "Bearer ")
		reqToken = splitToken[1]
	}
	userNumber := cookie.UserNum(`UserID`)
	userID := GetUserID(reqToken)
	newCtx := context.WithValue(ctx, userNumber, userID)
	return handler(newCtx, req)
}
