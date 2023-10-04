package config

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"net"
	"strings"
)

// Не экспортированная переменная AppAddr содержит адрес и порт для запуска сервера
var appAddr string
var shortURLAddr string
var linkFile string

const localHost = "http://localhost"
const localPort = "8080"

//var LinkFile = "internal/storage/links.json"

type Additional struct {
	Place     string
	OuterData string
	InnerData string
}

type OuterConfig struct {
	Default struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
	}
	Env struct {
		AppAddr      string `env:"SERVER_ADDRESS"`
		ShortURLAddr string `env:"BASE_URL"`
		LinkFile     string `env:"FILE_STORAGE_PATH"`
	}
	Flag struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
	}
	Final struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
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
	flag.StringVar(&linkFile, "f", "", "path to fila with links")

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
	if config.Final.LinkFile[0:1] == "/" {
		config.Final.LinkFile = config.Final.LinkFile[1:]
	}
	return err
}

func HandleConfig() error {

	err := env.Parse(&Config.Env)
	if err == nil {
		parseFlags()
		Config.Flag.AppAddr = appAddr
		Config.Flag.ShortURLAddr = shortURLAddr
		Config.Flag.LinkFile = linkFile

		Config.Default.AppAddr = fmt.Sprintf("%s:%s", localHost, localPort)
		Config.Default.ShortURLAddr = fmt.Sprintf("%s:%s", localHost, localPort)
		Config.Default.LinkFile = "internal/storage/links.json"

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

		if Config.Env.LinkFile != "" {
			Config.Final.LinkFile = Config.Env.LinkFile
		} else if Config.Flag.LinkFile != "" /*&& config.Env.ShortURLAddr == ""*/ {
			Config.Final.LinkFile = Config.Flag.LinkFile
		} else {
			Config.Final.LinkFile = Config.Default.LinkFile
		}

		err = Config.handleFinal()
	}
	return err
}
