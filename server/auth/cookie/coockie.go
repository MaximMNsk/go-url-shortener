package cookie

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"time"
)

func AuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		/** TODO
		 *  Get cookie
		 *  If cookie not set - set cookie
		 *  Decrypt data from cookie
		 *  If cookie not contain UserID - Unautorized answer
		 *  Else - serve
		 */
		token, err := r.Cookie("token")
		if err != nil && strings.Contains(err.Error(), `not present`) {
			logger.PrintLog(logger.WARN, err.Error())
			userID := 10
			newToken, err := BuildJWTString(userID)
			if err != nil {
				logger.PrintLog(logger.WARN, err.Error())
			}
			cookie := &http.Cookie{
				Name:    `token`,
				Value:   newToken,
				Expires: time.Now().Add(TOKEN_EXP),
				Path:    `/`,
			}
			http.SetCookie(w, cookie)
			additional := httpResp.Additional{}
			httpResp.Unauthorized(w, additional)
			return
			//fmt.Println(&cookie)
		}

		token, err = r.Cookie(`token`)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(token)
		remoteUserID := GetUserID(token.Value)

		if remoteUserID > 0 {
			next.ServeHTTP(w, r)
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

const TOKEN_EXP = time.Hour * 3
const SECRET_KEY = "supersecretkey"

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID int) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		// собственное утверждение
		UserID: 1,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
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
		return []byte(SECRET_KEY), nil
	})

	if err != nil {
		return -1
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return -1
	}

	fmt.Println("Token is valid")

	// возвращаем ID пользователя в читаемом виде
	return claims.UserID
}
