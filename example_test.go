package go_url_shortener_test

import (
	"context"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"github.com/carlmjohnson/requests"
	"io"
	"net/http"
	"strings"
	"time"
)

// HttpClient - структура объекта клиента
type HttpClient struct {
	host string
}

// MakeReq - выполняет http-запрос с указанными параметрами
func (cl *HttpClient) MakeReq(method string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, cl.host, body)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}

	client := http.DefaultClient
	client.CheckRedirect = requests.NoFollow

	response, err := client.Do(request)
	if err != nil {
		fmt.Println(`Request error: `, err.Error())
	}
	return response, err
}

func Example() {

	// Инициализируем конфиг.
	var conf config.OuterConfig
	err := conf.InitConfig(true)
	conf.Final.AppAddr = `localhost:8181`
	conf.Final.ShortURLAddr = `http://localhost:8181`
	if err != nil {
		fmt.Println(`Config error: `, err.Error())
	}

	ctx := context.Background()

	// Запускаем сервер.
	var serv server.Server
	serv.Init(conf, false)
	go func() {
		err = serv.Start(ctx)
		if err != nil {
			fmt.Println(`Starting error: `, err.Error())
		}
	}()

	// Создаем клиент
	client := &HttpClient{
		host: `http://localhost:8181/`,
	}

	// Подождем пока сервер запустится в отдельной горутине
	time.Sleep(100 * time.Millisecond)

	// Инициализируем и выполняем ping запрос.
	host := client.host
	client.host = client.host + `ping`
	response, err := client.MakeReq(http.MethodGet, nil)
	client.host = host

	fmt.Println(response.StatusCode)

	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Инициализируем и выполняем post запрос.
	body := strings.NewReader(`ya.ru`)

	response, err = client.MakeReq(http.MethodPost, body)
	fmt.Println(response.StatusCode)

	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Инициализируем и выполняем get запрос.
	shortLinkID := sha1hash.Create(`ya.ru`, 8)
	client.host = client.host + shortLinkID

	response, err = client.MakeReq(http.MethodGet, nil)

	fmt.Println(response.StatusCode)
	fmt.Println(response.Header.Get(`Location`))
	err = response.Body.Close()
	if err != nil {
		fmt.Println(`Close body error: `, err.Error())
	}

	// Останавливаем сервер.
	err = serv.Stop(ctx)
	if err != nil {
		fmt.Println(`Server error: `, err.Error())
	}

	// Unordered output:
	// 200
	// 201
	// 307
	// ya.ru
}
