package util

import (
	"os"
	"path/filepath"
	"strings"
	"fmt"
)

type BuildableService struct {
	Dir        string
	Dockerfile string
	Name       string
}

//FindServices finds all of the buildable services in the given directory
//(e.g., folders which contain a Dockerfile)
func FindServices(dir string, ignorePaths []string) ([]BuildableService, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var ret []BuildableService

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		for _, ignorePath := range ignorePaths {
			if filepath.Clean(filepath.Join(dir, ignorePath)) == path {
				return filepath.SkipDir
			}
		}

		if info.Name() == "Dockerfile" {
			ret = append(ret, BuildableService{
				Dir:        filepath.Dir(path),
				Dockerfile: "Dockerfile",
				Name:       filepath.Base(filepath.Dir(path)),
			})
		} else if strings.HasSuffix(info.Name(), ".Dockerfile") {
			name := filepath.Base(filepath.Dir(path)) + "-" + strings.TrimSuffix(info.Name(), ".Dockerfile")
			ret = append(ret, BuildableService{
				Dir:        filepath.Dir(path),
				Dockerfile: info.Name(),
				Name:       name,
			})
		}
		return nil
	})

	return ret, err
}
