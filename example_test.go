package go_url_shortener_test

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"github.com/carlmjohnson/requests"
	"net/http"
	"strings"
)

func Example() {

	// Инициализируем конфиг.
	var conf config.OuterConfig
	_ = conf.InitConfig(true)

	// Запускаем сервер.
	var serv server.Server
	serv.Config = conf
	_ = serv.Start()

	// Инициализируем и выполняем ping запрос.
	request, _ := http.NewRequest(http.MethodGet, `http://localhost:8080/ping`, nil)
	response, _ := http.DefaultClient.Do(request)
	fmt.Println(response.StatusCode)

	// Инициализируем и выполняем post запрос.
	body := strings.NewReader(`ya.ru`)
	request, _ = http.NewRequest(http.MethodPost, `http://localhost:8080/`, body)
	response, _ = http.DefaultClient.Do(request)
	fmt.Println(response.StatusCode)
	_ = response.Body.Close()

	// Инициализируем и выполняем get запрос.
	shortLinkID := sha1hash.Create(`ya.ru`, 8)
	request, _ = http.NewRequest(http.MethodGet, `http://localhost:8080/`+shortLinkID, nil)
	client := http.DefaultClient
	client.CheckRedirect = requests.NoFollow
	response, _ = client.Do(request)
	fmt.Println(response.StatusCode)
	fmt.Println(response.Header.Get(`Location`))

	// Output:
	// 200
	// 201
	// 307
	// ya.ru

	// Останавливаем сервер.
	_ = serv.Stop()
}
