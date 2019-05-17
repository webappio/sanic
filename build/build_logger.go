package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

//Logger takes log messages from the buildkit build server(s) and stores them
type Logger interface {
	ProcessStatus(service string, status *client.SolveStatus) error
	Close()
}

type flatfileLogger struct {
	LogDirectory string

	openFiles map[string]*os.File
}

//NewFlatfileLogger builds a new Logger which writes text logs to (repository root)/logs/(service name).log
func NewFlatfileLogger(logDirectory string) Logger {
	return &flatfileLogger{
		LogDirectory: logDirectory,
		openFiles:    make(map[string]*os.File),
	}
}

func (logger *flatfileLogger) ProcessStatus(service string, status *client.SolveStatus) error {
	var logFile *os.File

	if existingFile, ok := logger.openFiles[service]; ok {
		logFile = existingFile
	} else {
		err := os.MkdirAll(logger.LogDirectory, 0700)
		if err != nil {
			return errors.Errorf(
				"Could not make the logs output directory at %s: %s",
				logger.LogDirectory,
				err.Error())
		}
		logFile, err = os.OpenFile(
			filepath.Join(logger.LogDirectory, service+".log"),
			os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		logger.openFiles[service] = logFile
	}
	for _, log := range status.Logs {
		logFile.WriteString(fmt.Sprintf("[%s] ", log.Timestamp))
		_, err := logFile.Write(log.Data)
		if err != nil {
			return errors.Errorf(
				"Could not write to %s's logs: %s",
				service,
				err.Error())
		}
	}
	return nil
}

func (logger *flatfileLogger) Close() {
	for _, f := range logger.openFiles {
		f.Close()
	}
}
