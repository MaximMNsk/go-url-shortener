// Package config - пакет для работы с параметрами конфигурации.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
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
		AppAddr       string
		ShortURLAddr  string
		LinkFile      string
		DB            string
		TrustedSubnet string
	}
	Env struct {
		IsSecure      bool   `env:"ENABLE_HTTPS"`
		AppAddr       string `env:"SERVER_ADDRESS"`
		ShortURLAddr  string `env:"BASE_URL"`
		LinkFile      string `env:"FILE_STORAGE_PATH"`
		DB            string `env:"DATABASE_DSN"`
		TrustedSubnet string `env:"TRUSTED_SUBNET"`
	}
	Flag struct {
		IsSecure      bool
		AppAddr       string
		ShortURLAddr  string
		LinkFile      string
		DB            string
		TrustedSubnet string
	}
	ConfFile struct {
		Path          string
		IsSecure      bool   `json:"enable_https"`
		AppAddr       string `json:"server_address"`
		ShortURLAddr  string `json:"base_url"`
		LinkFile      string `json:"file_storage_path"`
		DB            string `json:"database_dsn"`
		TrustedSubnet string `json:"trusted_subnet"`
	}
	Final struct {
		IsSecure      bool
		AppAddr       string
		ShortURLAddr  string
		LinkFile      string
		DB            string
		TrustedSubnet string
	}
}

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных.
func (config *OuterConfig) parseFlags() {
	if flag.Lookup(`a`) == nil {
		flag.StringVar(&config.Flag.AppAddr, "a", "", "address and port to run server")
	}
	if flag.Lookup(`b`) == nil {
		flag.StringVar(&config.Flag.ShortURLAddr, "b", "", "address and port to short link")
	}
	if flag.Lookup(`f`) == nil {
		flag.StringVar(&config.Flag.LinkFile, "f", "", "path to file with links")
	}
	if flag.Lookup(`d`) == nil {
		flag.StringVar(&config.Flag.DB, "d", "", "db connection")
	}
	if flag.Lookup(`c`) == nil {
		flag.StringVar(&config.ConfFile.Path, "c", "", "config file path")
	}
	if flag.Lookup(`s`) == nil {
		flag.BoolVar(&config.Flag.IsSecure, "s", false, "secure connection")
	}
	if flag.Lookup(`t`) == nil {
		flag.StringVar(&config.ConfFile.Path, "t", "", "trusted subnet")
	}
	flag.Parse()
}

// handleFinal финализирует подготовку объекта конфигурации.
// Выполняется после инициализации и получения параметров.
func (config *OuterConfig) handleFinal() error {
	r := strings.NewReplacer(
		proto, "",
		secProto, "",
	)
	config.Final.AppAddr = r.Replace(config.Final.AppAddr)

	aHost, aPort, err := net.SplitHostPort(config.Final.AppAddr)
	if err != nil {
		return err
	}

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
	config.Default.TrustedSubnet = "127.0.0.1/24"
	//Config.Default.DB = "user=postgres password=12345 dbname=postgres sslmode=disable"
	return err
}

// parseEnv обрабатывает переменные окружения
// и сохраняет их значения в соответствующих переменных.
func (config *OuterConfig) parseEnv() error {
	err := env.Parse(&config.Env)
	config.ConfFile.Path = os.Getenv(`CONFIG`)
	return err
}

// ParseConfigFile разбирает файл конфигурации
// и сохраняет данные соответствующей структуре.
func (config *OuterConfig) ParseConfigFile() error {
	data, err := os.ReadFile(config.ConfFile.Path)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return fmt.Errorf("config file is empty")
	}

	err = json.Unmarshal(data, &config.ConfFile)
	if err != nil {
		return err
	}

	return nil
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

		if len(config.ConfFile.Path) != 0 {
			err = config.ParseConfigFile()
			if err != nil {
				return err
			}
		}
	}

	switch {
	case config.Flag.IsSecure:
		config.Final.IsSecure = true
	case config.Env.IsSecure:
		config.Final.IsSecure = true
	case config.ConfFile.IsSecure:
		config.Final.IsSecure = true
	default:
		config.Final.IsSecure = config.Default.IsSecure
	}

	switch {
	case config.Flag.AppAddr != "":
		config.Final.AppAddr = config.Flag.AppAddr
	case config.Env.AppAddr != "":
		config.Final.AppAddr = config.Env.AppAddr
	case config.ConfFile.AppAddr != "":
		config.Final.AppAddr = config.ConfFile.AppAddr
	default:
		config.Final.AppAddr = config.Default.AppAddr
	}

	switch {
	case config.Flag.ShortURLAddr != "":
		config.Final.ShortURLAddr = config.Flag.ShortURLAddr
	case config.Env.ShortURLAddr != "":
		config.Final.ShortURLAddr = config.Env.ShortURLAddr
	case config.ConfFile.ShortURLAddr != "":
		config.Final.ShortURLAddr = config.ConfFile.ShortURLAddr
	default:
		config.Final.ShortURLAddr = config.Default.ShortURLAddr
	}

	switch {
	case config.Flag.LinkFile != "":
		config.Final.LinkFile = config.Flag.LinkFile
	case config.Env.LinkFile != "":
		config.Final.LinkFile = config.Env.LinkFile
	case config.ConfFile.LinkFile != "":
		config.Final.LinkFile = config.ConfFile.LinkFile
	default:
		config.Final.LinkFile = config.Default.LinkFile
	}

	switch {
	case config.Flag.DB != "":
		config.Final.DB = config.Flag.DB
	case config.Env.DB != "":
		config.Final.DB = config.Env.DB
	case config.ConfFile.DB != "":
		config.Final.DB = config.ConfFile.DB
	default:
		config.Final.DB = config.Default.DB
	}

	switch {
	case config.Flag.TrustedSubnet != "":
		config.Final.TrustedSubnet = config.Flag.TrustedSubnet
	case config.Env.TrustedSubnet != "":
		config.Final.TrustedSubnet = config.Env.TrustedSubnet
	case config.ConfFile.TrustedSubnet != "":
		config.Final.TrustedSubnet = config.ConfFile.TrustedSubnet
	default:
		config.Final.TrustedSubnet = config.Default.TrustedSubnet
	}

	err = config.handleFinal()
	return err
}
