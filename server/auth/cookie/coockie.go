package cookie

import (
	"context"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/randomizer"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//var UserID int

type UserNum string

func AuthSetter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("token")

		UserID, errUserID := randomizer.RandDigitalBytes(3)
		if errUserID != nil {
			logger.PrintLog(logger.WARN, err.Error())
		}

		if err != nil && strings.Contains(err.Error(), `not present`) {
			logInfo := fmt.Sprintf("Set userID: %d", UserID)
			logger.PrintLog(logger.INFO, logInfo)
			newToken, err := BuildJWTString(UserID)
			if err != nil {
				logger.PrintLog(logger.WARN, err.Error())
			}
			logger.PrintLog(logger.DEBUG, `Set token: `+newToken)
			cookie := &http.Cookie{
				Name:    `token`,
				Value:   newToken,
				Expires: time.Now().Add(TokenExp),
				Path:    `/`,
			}
			http.SetCookie(w, cookie)
		}
		userNumber := UserNum(`UserID`)
		ctx := context.WithValue(r.Context(), userNumber, strconv.Itoa(UserID))
		newReqCtx := r.WithContext(ctx)
		next.ServeHTTP(w, newReqCtx)
	})
}

func AuthChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("token")
		if err != nil {
			additional := httpResp.Additional{}
			httpResp.NoContent(w, additional)
			return
		}
		logger.PrintLog(logger.DEBUG, `Present token: `+token.Value)
		UserID := GetUserID(token.Value)
		logger.PrintLog(logger.DEBUG, `Present userID: `+strconv.Itoa(UserID))
		if UserID > 0 {
			userNumber := UserNum(`UserID`)
			ctx := context.WithValue(r.Context(), userNumber, UserID)
			newReqCtx := r.WithContext(ctx)
			next.ServeHTTP(w, newReqCtx)
			return
		}
		additional := httpResp.Additional{}
		httpResp.Unauthorized(w, additional)
	})
}

func AuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := r.Cookie("token")
		if err != nil && strings.Contains(err.Error(), `not present`) {
			logger.PrintLog(logger.WARN, err.Error())
			UserID, err := randomizer.RandDigitalBytes(3)
			logInfo := fmt.Sprintf("Set userID: %d", UserID)
			logger.PrintLog(logger.INFO, logInfo)
			if err != nil {
				logger.PrintLog(logger.WARN, err.Error())
			}
			newToken, err := BuildJWTString(UserID)
			if err != nil {
				logger.PrintLog(logger.WARN, err.Error())
			}
			logger.PrintLog(logger.DEBUG, `Set token: `+newToken)
			cookie := &http.Cookie{
				Name:    `token`,
				Value:   newToken,
				Expires: time.Now().Add(TokenExp),
				Path:    `/`,
			}
			http.SetCookie(w, cookie)
			userNumber := UserNum(`UserID`)
			ctx := context.WithValue(r.Context(), userNumber, UserID)
			newReqCtx := r.WithContext(ctx)
			next.ServeHTTP(w, newReqCtx)
			//next.ServeHTTP(w, r)
			return
		}

		logger.PrintLog(logger.INFO, "Token: "+token.Value)
		UserID := GetUserID(token.Value)

		if UserID > 0 {
			userNumber := UserNum(`UserID`)
			ctx := context.WithValue(r.Context(), userNumber, UserID)
			newReqCtx := r.WithContext(ctx)
			next.ServeHTTP(w, newReqCtx)
			return
		}
		additional := httpResp.Additional{}
		httpResp.Unauthorized(w, additional)
	})
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 48
const SecretKey = "superPuperSecretKey"

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}
func GetUserID(tokenString string) int {
	// создаём экземпляр структуры с утверждениями
	claims := &Claims{}
	// парсим из строки токена tokenString в структуру claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(SecretKey), nil
	})

	if err != nil {
		return -1
	}

	if !token.Valid {
		//fmt.Println("Token is not valid")
		logger.PrintLog(logger.WARN, `Invalid token`)
		return -1
	}

	//fmt.Println("Token is valid")
	logger.PrintLog(logger.INFO, `Token is valid!`)

	// возвращаем ID пользователя в читаемом виде
	return claims.UserID
}
