package logger

import (
	"log"
	"os"
)

// INFO уровень логирования.
const INFO = "INFO"

// DEBUG уровень логирования.
const DEBUG = "DEBUG"

// WARN уровень логирования.
const WARN = "WARN"

// ERROR уровень логирования.
const ERROR = "ERROR"

// FATAL уровень логирования.
const FATAL = "FATAL"

// PrintLog - формирует форматированный вывод лога.
func PrintLog(level string, message string) {
	switch level {
	case INFO:
		logger := log.New(os.Stdout, INFO+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case DEBUG:
		logger := log.New(os.Stdout, DEBUG+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case WARN:
		logger := log.New(os.Stdout, WARN+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case ERROR:
		logger := log.New(os.Stdout, ERROR+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case FATAL:
		logger := log.New(os.Stdout, FATAL+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	default:
		logger := log.New(os.Stdout, "USER_MESSAGE: ", log.Ldate|log.Ltime)
		logger.Println(message)
	}
}
