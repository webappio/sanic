package util

import (
	"github.com/distributed-containers-inc/sanic/shell"
	"os"
	"path/filepath"
)

//FindServices finds all of the buildable services in the sanic root directory
//(e.g., folders which contain a Dockerfile)
func FindServices() ([]string, error) {
	s, err := shell.Current()
	if err != nil {
		return nil, err
	}

	var ret []string

	err = filepath.Walk(s.GetSanicRoot(), func(path string, info os.FileInfo, err error) error {
		if info.Name() == "Dockerfile" {
			ret = append(ret, filepath.Dir(path))
		}
		return nil
	})

	return ret, err
}
