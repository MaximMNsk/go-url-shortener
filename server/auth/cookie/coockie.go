// Package cookie - работа с куки.
// Установка куки и авторизация по информации из куки.
// Полноценная работа с JWT.
package cookie

import (
	"context"
	"errors"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/randomizer"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"time"
)

// UserNum - тип для номера пользователя.
type UserNum string

// AuthSetter - устанавливает куки для авторизованного пользователя.
func AuthSetter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("token")

		UserID, errUserID := randomizer.RandDigitalBytes(9)
		if errUserID != nil {
			logger.PrintLog(logger.WARN, err.Error(), true)
		}

		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				newToken, err := BuildJWTString(UserID)
				if err != nil {
					httpResp.BadRequest(w)
					return
				}
				cookie := &http.Cookie{
					Name:    `token`,
					Value:   newToken,
					Expires: time.Now().Add(tokenExp),
					Path:    `/`,
				}
				http.SetCookie(w, cookie)
			} else {
				httpResp.BadRequest(w)
				return
			}
		}

		userNumber := UserNum(`UserID`)
		ctx := context.WithValue(r.Context(), userNumber, UserID)
		newReqCtx := r.WithContext(ctx)
		next.ServeHTTP(w, newReqCtx)
	})
}

// AuthChecker - проверяет куки пользователя.
func AuthChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("token")
		if err != nil {
			additional := httpResp.Additional{}
			httpResp.Unauthorized(w, additional)
			return
		}
		UserID := GetUserID(token.Value)
		if UserID < 0 {
			additional := httpResp.Additional{}
			httpResp.Unauthorized(w, additional)
			return
		}
		userNumber := UserNum(`UserID`)
		ctx := context.WithValue(r.Context(), userNumber, UserID)
		newReqCtx := r.WithContext(ctx)
		next.ServeHTTP(w, newReqCtx)
		return
	})
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const tokenExp = time.Hour * 48
const secretKey = "superPuperSecretKey"

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExp)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// GetUserID - получает UserID из токена.
func GetUserID(tokenString string) int {
	// создаём экземпляр структуры с утверждениями
	claims := &Claims{}
	// парсим из строки токена tokenString в структуру claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return -1
	}

	if !token.Valid {
		logger.PrintLog(logger.WARN, `Invalid token`, true)
		return -1
	}

	// возвращаем ID пользователя в читаемом виде
	return claims.UserID
}
