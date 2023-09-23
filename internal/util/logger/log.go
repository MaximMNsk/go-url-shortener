package logger

import (
	"log"
	"os"
)

const INFO = "INFO"
const DEBUG = "DEBUG"
const WARN = "WARN"
const ERROR = "ERROR"
const FATAL = "FATAL"

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
		logger.Fatalln(message)
	case FATAL:
		logger := log.New(os.Stdout, FATAL+": ", log.Ldate|log.Ltime)
		logger.Fatalln(message)
	default:
		logger := log.New(os.Stdout, "USER_MESSAGE: ", log.Ldate|log.Ltime)
		logger.Println(message)
	}
}
