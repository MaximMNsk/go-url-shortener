package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"net"
	"strings"
)

// неэкспортированная переменная AppAddr содержит адрес и порт для запуска сервера
var appAddr string
var shortURLAddr string

const localHost = "http://localhost"
const localPort = "8080"
const LinkFile = "internal/storage/links.json"

type Additional struct {
	Place     string
	OuterData string
	InnerData string
}

type OuterConfig struct {
	Default struct {
		AppAddr      string
		ShortURLAddr string
	}
	Env struct {
		AppAddr      string `env:"SERVER_ADDRESS"`
		ShortURLAddr string `env:"BASE_URL"`
	}
	Flag struct {
		AppAddr      string
		ShortURLAddr string
	}
	Final struct {
		AppAddr      string
		ShortURLAddr string
	}
}

var Config OuterConfig

/**
 * Config handlers
 */

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&appAddr, "a", "", "address and port to run server")
	flag.StringVar(&shortURLAddr, "b", "", "address and port to short link")

	flag.Parse()
}

func (config *OuterConfig) handleFinal() error {
	config.Final.AppAddr = strings.Replace(config.Final.AppAddr, "http://", "", -1)
	aHost, aPort, err := net.SplitHostPort(config.Final.AppAddr)
	if err == nil {
		if aHost == "" {
			config.Final.AppAddr = "localhost:" + aPort
		}

		if config.Final.ShortURLAddr[0:7] != "http://" {
			config.Final.ShortURLAddr = "http://" + config.Final.ShortURLAddr
		}
	}
	return err
}

func HandleConfig() error {

	err := env.Parse(&Config.Env)
	if err == nil {
		parseFlags()
		Config.Flag.AppAddr = appAddr
		Config.Flag.ShortURLAddr = shortURLAddr

		Config.Default.AppAddr = fmt.Sprintf("%s:%s", localHost, localPort)
		Config.Default.ShortURLAddr = fmt.Sprintf("%s:%s", localHost, localPort)

		if Config.Env.AppAddr != "" {
			Config.Final.AppAddr = Config.Env.AppAddr
		} else if Config.Flag.AppAddr != "" /*&& config.Env.AppAddr == ""*/ {
			Config.Final.AppAddr = Config.Flag.AppAddr
		} else {
			Config.Final.AppAddr = Config.Default.AppAddr
		}

		if Config.Env.ShortURLAddr != "" {
			Config.Final.ShortURLAddr = Config.Env.ShortURLAddr
		} else if Config.Flag.ShortURLAddr != "" /*&& config.Env.ShortURLAddr == ""*/ {
			Config.Final.ShortURLAddr = Config.Flag.ShortURLAddr
		} else {
			Config.Final.ShortURLAddr = Config.Default.ShortURLAddr
		}
		err = Config.handleFinal()
	}
	return err
}
