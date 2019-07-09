package util

import (
	"os"
	"path/filepath"
)

//FindServices finds all of the buildable services in the sanic root directory
//(e.g., folders which contain a Dockerfile)
func FindServices(dir string) ([]string, error) {
	var ret []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "Dockerfile" {
			ret = append(ret, filepath.Dir(path))
		}
		return nil
	})

	return ret, err
}
