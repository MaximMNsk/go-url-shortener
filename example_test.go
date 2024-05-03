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
	err := conf.InitConfig(true)
	if err != nil {
		fmt.Println(`Config error: `, err.Error())
	}

	// Запускаем сервер.
	var serv *server.Server
	serv.Config = conf
	go func(*server.Server) {
		_ = serv.Start()
	}(serv)

	// Инициализируем и выполняем ping запрос.
	request, err := http.NewRequest(http.MethodGet, `http://localhost:8080/ping`, nil)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	fmt.Println(response.StatusCode)
	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Инициализируем и выполняем post запрос.
	body := strings.NewReader(`ya.ru`)
	request, err = http.NewRequest(http.MethodPost, `http://localhost:8080/`, body)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	fmt.Println(response.StatusCode)
	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Инициализируем и выполняем get запрос.
	shortLinkID := sha1hash.Create(`ya.ru`, 8)
	request, err = http.NewRequest(http.MethodGet, `http://localhost:8080/`+shortLinkID, nil)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	client := http.DefaultClient
	client.CheckRedirect = requests.NoFollow
	response, err = client.Do(request)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	fmt.Println(response.StatusCode)
	fmt.Println(response.Header.Get(`Location`))
	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Останавливаем сервер.
	err = serv.Stop()
	if err != nil {
		fmt.Println(`Server error: `, err.Error())
	}

	// Unordered output:
	// 200
	// 201
	// 307
	// ya.ru
}
