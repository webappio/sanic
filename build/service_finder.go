package build

import (
	"os"
	"path/filepath"
)

func FindServices(path string) ([]string, error) {
	var ret []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "Dockerfile" {
			ret = append(ret, path)
		}

		return nil
	})

	return ret, err
}