package pkg1

import "os"

func someFunc() {
	os.Exit(0) // want "прямой вызов os.Exit в функции main пакета main запрещен"
}
