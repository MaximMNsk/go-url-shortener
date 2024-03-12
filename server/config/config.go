package config

import (
	"flag"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/pathhandler"
	"github.com/caarlos0/env/v6"
	"net"
	"path/filepath"
	"strings"
)

const localHost = "http://localhost"
const localPort = "8080"

type OuterConfig struct {
	Default struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
	Env struct {
		AppAddr      string `env:"SERVER_ADDRESS"`
		ShortURLAddr string `env:"BASE_URL"`
		LinkFile     string `env:"FILE_STORAGE_PATH"`
		DB           string `env:"DATABASE_DSN"`
	}
	Flag struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
	Final struct {
		AppAddr      string
		ShortURLAddr string
		LinkFile     string
		DB           string
	}
}

/** TODO delete global var. Make methods from all func */
//var Config OuterConfig

/**
 * Config handlers
 */

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func (config *OuterConfig) parseFlags() {
	flag.StringVar(&config.Flag.AppAddr, "a", "", "address and port to run server")
	flag.StringVar(&config.Flag.ShortURLAddr, "b", "", "address and port to short link")
	flag.StringVar(&config.Flag.LinkFile, "f", "", "path to file with links")
	flag.StringVar(&config.Flag.DB, "d", "", "db connection")
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
	config.Final.LinkFile = filepath.Join(config.Final.LinkFile)

	return err
}

func (config *OuterConfig) setDefaults() error {
	config.Default.AppAddr = fmt.Sprintf("%s:%s", localHost, localPort)
	config.Default.ShortURLAddr = fmt.Sprintf("%s:%s", localHost, localPort)
	rootPath, err := pathhandler.ProjectRoot()
	config.Default.LinkFile = filepath.Join(rootPath, "internal/storage/files/links.json")
	config.Default.DB = "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	//Config.Default.DB = "user=postgres password=12345 dbname=postgres sslmode=disable"
	return err
}

func (config *OuterConfig) parseEnv() error {
	err := env.Parse(&config.Env)
	return err
}

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

	if config.Env.AppAddr != "" {
		config.Final.AppAddr = config.Env.AppAddr
	} else if config.Flag.AppAddr != "" {
		config.Final.AppAddr = config.Flag.AppAddr
	} else {
		config.Final.AppAddr = config.Default.AppAddr
	}

	if config.Env.ShortURLAddr != "" {
		config.Final.ShortURLAddr = config.Env.ShortURLAddr
	} else if config.Flag.ShortURLAddr != "" {
		config.Final.ShortURLAddr = config.Flag.ShortURLAddr
	} else {
		config.Final.ShortURLAddr = config.Default.ShortURLAddr
	}

	if config.Env.LinkFile != "" {
		config.Final.LinkFile = config.Env.LinkFile
	} else if config.Flag.LinkFile != "" {
		config.Final.LinkFile = config.Flag.LinkFile
	} else {
		config.Final.LinkFile = config.Default.LinkFile
	}

	if config.Env.DB != "" {
		config.Final.DB = config.Env.DB
	} else if config.Flag.DB != "" {
		config.Final.DB = config.Flag.DB
	} else {
		config.Final.DB = config.Default.DB
	}

	err = config.handleFinal()
	return err
}
