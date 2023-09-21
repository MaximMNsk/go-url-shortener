package logger

import (
	"log"
	"os"
)

func PrintLog(level string, message string) {
	switch level {
	case "INFO":
		logger := log.New(os.Stdout, level+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case "DEBUG":
		logger := log.New(os.Stdout, level+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case "WARN":
		logger := log.New(os.Stdout, level+": ", log.Ldate|log.Ltime)
		logger.Println(message)
	case "ERROR":
		logger := log.New(os.Stdout, level+": ", log.Ldate|log.Ltime)
		logger.Fatalln(message)
	case "FATAL":
		logger := log.New(os.Stdout, level+": ", log.Ldate|log.Ltime)
		logger.Fatalln(message)
	default:
		logger := log.New(os.Stdout, "USER_MESSAGE: ", log.Ldate|log.Ltime)
		logger.Println(message)
	}
}
