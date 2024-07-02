// Package pathhandler - пакет для поиска корневой директории проекта.
package pathhandler

import (
	"os"
	"path/filepath"
)

// ProjectRoot - возвращает корневую директорию проекта и ошибку.
func ProjectRoot() (string, error) {
	currentPath := ""
	var err error

	for i := 0; i < 9; i++ {
		currentPath, err = os.Getwd()
		if err == nil {
			modFile := filepath.Join(currentPath, "go.mod")
			if _, err = os.Stat(modFile); err == nil {
				break
			}
			err = os.Chdir("../")
		}
	}

	return currentPath, err
}
