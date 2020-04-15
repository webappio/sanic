package build

import (
	"fmt"
	"github.com/pkg/errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//Logger takes log messages from the buildkit build server(s) and stores them
type Logger interface {
	Log(service string, when time.Time, message ...interface{}) error
	Close()
	AddLogLineListener(func(service, logLine string))
}

type flatfileLogger struct {
	mutex              sync.Mutex
	LogDirectory       string
	currVertexStatuses map[string]string
	openFiles          map[string]*os.File
	logLineListeners   []func(service, logLine string)
	verbose            bool
}

//NewFlatfileLogger builds a new Logger which writes text logs to (repository root)/logs/(service name).log
func NewFlatfileLogger(logDirectory string, verbose bool) Logger {
	return &flatfileLogger{
		LogDirectory:       logDirectory,
		openFiles:          make(map[string]*os.File),
		currVertexStatuses: make(map[string]string),
		logLineListeners:   []func(service, logLine string){},
		verbose:            verbose,
	}
}

func (logger *flatfileLogger) logFile(service string) (*os.File, error) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	var logFile *os.File

	if existingFile, ok := logger.openFiles[service]; ok {
		logFile = existingFile
	} else {
		err := os.MkdirAll(logger.LogDirectory, 0700)
		if err != nil {
			return nil, errors.Errorf(
				"Could not make the logs output directory at %s: %s",
				logger.LogDirectory,
				err.Error())
		}
		logFile, err = os.OpenFile(
			filepath.Join(logger.LogDirectory, service+".log"),
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return nil, err
		}
		logFile.WriteString("") //wipe old logs
		logger.openFiles[service] = logFile
	}
	return logFile, nil
}

func (logger *flatfileLogger) Log(service string, when time.Time, message ...interface{}) error {
	f, err := logger.logFile(service)
	if err != nil {
		return err
	}
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	messageString := strings.Trim(fmt.Sprint(message...), "\r\n")
	_, err = f.WriteString(fmt.Sprintf("[%s] %s\n", when.In(time.Local), messageString))
	for _, listener := range logger.logLineListeners {
		listener(service, messageString+"\n")
	}
	if err != nil {
		return err
	}
	return nil
}

func humanReadableBytes(bytes int64) string {
	suf := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	if bytes == 0 {
		return fmt.Sprintf("0%s", suf[0])
	}
	place := math.Logb(math.Abs(float64(bytes))) / 10
	return fmt.Sprintf("%.2f%s", float64(bytes)/math.Pow(1024, math.Floor(place)), suf[int64(place)])
}

func (logger *flatfileLogger) Close() {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	for _, f := range logger.openFiles {
		f.Close()
	}
}

func (logger *flatfileLogger) AddLogLineListener(processLog func(service, logLine string)) {
	logger.logLineListeners = append(logger.logLineListeners, processLog)
}
