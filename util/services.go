package util

import (
	"os"
	"path/filepath"
)

//FindServices finds all of the buildable services in the given directory
//(e.g., folders which contain a Dockerfile)
func FindServices(dir string, ignorePaths []string) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var ret []string

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		for _, ignorePath := range ignorePaths {
			if filepath.Clean(filepath.Join(dir, ignorePath)) == path {
				return filepath.SkipDir
			}
		}

		if info.Name() == "Dockerfile" {
			ret = append(ret, filepath.Dir(path))
		}
		return nil
	})

	return ret, err
}
