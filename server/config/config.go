// Package config - пакет для работы с параметрами конфигурации.
package config

import (
	"flag"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v6"

	"github.com/MaximMNsk/go-url-shortener/internal/util/pathhandler"
)

const localHost = "localhost"
const localPort = "8080"
const secProto = `https://`
const proto = `http://`

// OuterConfig - структура объекта конфигурации.
type OuterConfig struct {
	Default struct {
		IsSecure bool
		Cert     struct {
			KeyFile  string
			CertFile string
		}
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
	Env struct {
		IsSecure     bool   `env:"ENABLE_HTTPS"`
		AppAddr      string `env:"SERVER_ADDRESS"`
		ShortURLAddr string `env:"BASE_URL"`
		LinkFile     string `env:"FILE_STORAGE_PATH"`
		DB           string `env:"DATABASE_DSN"`
	}
	Flag struct {
		IsSecure     bool
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
	Final struct {
		IsSecure     bool
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
}

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func (config *OuterConfig) parseFlags() {
	flag.StringVar(&config.Flag.AppAddr, "a", "", "address and port to run server")
	flag.StringVar(&config.Flag.ShortURLAddr, "b", "", "address and port to short link")
	flag.StringVar(&config.Flag.LinkFile, "f", "", "path to file with links")
	flag.StringVar(&config.Flag.DB, "d", "", "db connection")
	flag.BoolVar(&config.Flag.IsSecure, "s", false, "secure connection")
	flag.Parse()
}

// handleFinal финализирует подготовку объекта конфигурации.
// Выполняется после инициализации и получения параметров.
func (config *OuterConfig) handleFinal() error {
	config.Final.AppAddr = strings.Replace(config.Final.AppAddr, proto, "", -1)
	config.Final.AppAddr = strings.Replace(config.Final.AppAddr, secProto, "", -1)
	aHost, aPort, err := net.SplitHostPort(config.Final.AppAddr)
	if err == nil {
		if aHost == "" {
			config.Final.AppAddr = "localhost:" + aPort
		}

		if config.Final.ShortURLAddr[0:4] != `http` {
			if config.Final.IsSecure {
				config.Final.ShortURLAddr = secProto + config.Final.ShortURLAddr
			} else {
				config.Final.ShortURLAddr = proto + config.Final.ShortURLAddr
			}
		}
	}
	config.Final.LinkFile = filepath.Join(config.Final.LinkFile)

	return err
}

// setDefaults - устанавливает умолчательные значения.
func (config *OuterConfig) setDefaults() error {
	config.Default.IsSecure = false
	config.Default.AppAddr = fmt.Sprintf("%s:%s", localHost, localPort)
	config.Default.ShortURLAddr = fmt.Sprintf("%s:%s", localHost, localPort)
	rootPath, err := pathhandler.ProjectRoot()
	config.Default.Cert.CertFile = filepath.Join(rootPath, "cmd/shortener/secure/certificate.crt")
	config.Default.Cert.KeyFile = filepath.Join(rootPath, "cmd/shortener/secure/privateKey.key")
	config.Default.LinkFile = filepath.Join(rootPath, "internal/storage/files/links.json")
	config.Default.DB = "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	//Config.Default.DB = "user=postgres password=12345 dbname=postgres sslmode=disable"
	return err
}

// parseEnv обрабатывает переменные окружения
// и сохраняет их значения в соответствующих переменных.
func (config *OuterConfig) parseEnv() error {
	err := env.Parse(&config.Env)
	return err
}

// InitConfig - инициализация объекта конфигурации.
func (config *OuterConfig) InitConfig(testMode bool) error {

	err := config.setDefaults()
	if err != nil {
		return err
	}
	if !testMode {
		err = config.parseEnv()
		if err != nil {
			return err
		}
		config.parseFlags()
	}

	switch {
	case config.Env.IsSecure:
		config.Final.IsSecure = true
	case config.Flag.IsSecure:
		config.Final.IsSecure = true
	default:
		config.Final.IsSecure = config.Default.IsSecure
	}

	switch {
	case config.Env.AppAddr != "":
		config.Final.AppAddr = config.Env.AppAddr
	case config.Flag.AppAddr != "":
		config.Final.AppAddr = config.Flag.AppAddr
	default:
		config.Final.AppAddr = config.Default.AppAddr
	}

	switch {
	case config.Env.ShortURLAddr != "":
		config.Final.ShortURLAddr = config.Env.ShortURLAddr
	case config.Flag.ShortURLAddr != "":
		config.Final.ShortURLAddr = config.Flag.ShortURLAddr
	default:
		config.Final.ShortURLAddr = config.Default.ShortURLAddr
	}

	switch {
	case config.Env.LinkFile != "":
		config.Final.LinkFile = config.Env.LinkFile
	case config.Flag.LinkFile != "":
		config.Final.LinkFile = config.Flag.LinkFile
	default:
		config.Final.LinkFile = config.Default.LinkFile
	}

	switch {
	case config.Env.DB != "":
		config.Final.DB = config.Env.DB
	case config.Flag.DB != "":
		config.Final.DB = config.Flag.DB
	default:
		config.Final.DB = config.Default.DB
	}

	err = config.handleFinal()
	return err
}
