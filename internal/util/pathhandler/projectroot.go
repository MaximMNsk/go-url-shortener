package pathhandler

import (
	"os"
	"path/filepath"
)

func ProjectRoot() (string, error) {
	currentPath := ""
	var err error

	for i := 0; i < 9; i++ {
		currentPath, err = os.Getwd()
		if err == nil {
			modFile := filepath.Join(currentPath, "go.mod")
			if _, err = os.Stat(modFile); err == nil {
				break
			} else {
				err = os.Chdir("../")
				continue
			}
		}
	}

	return currentPath, err
}
