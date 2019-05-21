package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"time"
)

//Logger takes log messages from the buildkit build server(s) and stores them
type Logger interface {
	Log(service string, when time.Time, message ...interface{}) error
	ProcessStatus(service string, status *client.SolveStatus) error
	Close()
	AddLogLineListener(func(service, logLine string))
}

type flatfileLogger struct {
	LogDirectory string

	openFiles        map[string]*os.File
	logLineListeners []func(service, logLine string)
}

//NewFlatfileLogger builds a new Logger which writes text logs to (repository root)/logs/(service name).log
func NewFlatfileLogger(logDirectory string) Logger {
	return &flatfileLogger{
		LogDirectory:     logDirectory,
		openFiles:        make(map[string]*os.File),
		logLineListeners: []func(service, logLine string){},
	}
}

func (logger *flatfileLogger) Log(service string, when time.Time, message ...interface{}) error {
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
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		logFile.WriteString("") //wipe old logs
		logger.openFiles[service] = logFile
	}
	messageString := fmt.Sprint(message...)
	_, err := logFile.WriteString(fmt.Sprintf("[%s] %s\n", when.In(time.Local), messageString))
	for _, listener := range logger.logLineListeners {
		listener(service, messageString+"\n")
	}
	if err != nil {
		return err
	}
	return nil
}

func (logger *flatfileLogger) ProcessStatus(service string, status *client.SolveStatus) error {
	for _, log := range status.Logs {
		logMessage := []rune(string(log.Data))
		if logMessage[len(logMessage)-1] == '\n' {
			logMessage = logMessage[:len(logMessage)-1]
		}
		err := logger.Log(service, log.Timestamp, string(logMessage))
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

func (logger *flatfileLogger) AddLogLineListener(processLog func(service, logLine string)) {
	logger.logLineListeners = append(logger.logLineListeners, processLog)
}
