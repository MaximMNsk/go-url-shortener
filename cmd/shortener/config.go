package main

import (
	"flag"
)

// неэкспортированная переменная AppAddr содержит адрес и порт для запуска сервера
var AppAddr string
var ShortURLAddr string

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&AppAddr, "a", "", "address and port to run server")
	flag.StringVar(&ShortURLAddr, "b", "", "address and port to short link")

	flag.Parse()
}
