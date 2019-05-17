package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

type Logger interface {
	ProcessStatus(service string, status *client.SolveStatus) error
	Close()
}

type FlatfileLogger struct {
	LogDirectory string

	openFiles map[string]*os.File
}

func (logger FlatfileLogger) ProcessStatus(service string, status *client.SolveStatus) error {
	var logFile *os.File

	if logger.openFiles == nil {
		logger.openFiles = make(map[string]*os.File)
	}

	if existingFile, ok := logger.openFiles[service]; ok {
		logFile = existingFile
	} else {
		err := os.MkdirAll(logger.LogDirectory, 0700)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"Could not make the logs output directory at %s: %s",
				logger.LogDirectory,
				err.Error()))
		}
		logFile, err = os.OpenFile(
			filepath.Join(logger.LogDirectory, service+".log"),
			os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		logger.openFiles[service] = logFile
	}
	for _, log := range status.Logs {
		logFile.WriteString(fmt.Sprintf("[%s] ", log.Timestamp))
		_, err := logFile.Write(log.Data)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"Could not write to %s's logs: %s",
				service,
				err.Error()))
		}
	}
	return nil
}

func (logger FlatfileLogger) Close() {
	for _, f := range logger.openFiles {
		f.Close()
	}
}
