package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
	"github.com/pkg/errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	mutex sync.Mutex

	LogDirectory string

	currVertexStatuses map[string]string
	lastStatusLogSize  int64

	openFiles        map[string]*os.File
	logLineListeners []func(service, logLine string)
}

//NewFlatfileLogger builds a new Logger which writes text logs to (repository root)/logs/(service name).log
func NewFlatfileLogger(logDirectory string) Logger {
	return &flatfileLogger{
		LogDirectory:       logDirectory,
		openFiles:          make(map[string]*os.File),
		currVertexStatuses: make(map[string]string),
		logLineListeners:   []func(service, logLine string){},
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
	numBytes, err := f.WriteString(fmt.Sprintf("[%s] %s\n", when.In(time.Local), messageString))
	for _, listener := range logger.logLineListeners {
		listener(service, messageString+"\n")
	}
	if err != nil {
		return err
	}
	logger.lastStatusLogSize -= int64(numBytes)
	if logger.lastStatusLogSize < 0 {
		logger.lastStatusLogSize = 0
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

func (logger *flatfileLogger) logStatus(service string, status *client.VertexStatus) error {
	f, err := logger.logFile(service)
	if err != nil {
		return err
	}

	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	if logger.lastStatusLogSize > 0 {
		_, err = f.Seek(-logger.lastStatusLogSize, io.SeekEnd)
	}
	if err != nil {
		return err
	}

	var idText string
	if strings.HasPrefix(status.ID, "sha256:") {
		idText = status.ID[7:19]
	} else {
		idText = status.ID
	}

	var statusText string
	if status.Total != 0 {
		statusText = fmt.Sprintf("%s %s/%s", idText, humanReadableBytes(status.Current), humanReadableBytes(status.Total))
	}  else {
		statusText = fmt.Sprintf("%s %s", idText, humanReadableBytes(status.Current))
	}
	statusTextTimestamp := fmt.Sprintf("[%s]: %s", status.Timestamp.In(time.Local), statusText)

	if status.Completed != nil {
		delete(logger.currVertexStatuses, status.ID)
		_, err = f.WriteString(statusTextTimestamp+"\n")
	} else {
		logger.currVertexStatuses[status.ID] = statusTextTimestamp
	}
	var statuses []string
	for _, s := range logger.currVertexStatuses {
		statuses = append(statuses, s+"\n")
	}
	sort.Strings(statuses)
	written, err := f.WriteString(strings.Join(statuses, ""))
	if err != nil {
		return err
	}
	logger.lastStatusLogSize = int64(written)
	for _, listener := range logger.logLineListeners {
		if len(statuses) > 0 {
			statusText = statuses[len(statuses)-1]
			statusText = statusText[strings.Index(statusText, "]")+3:] //TODO HACK
		}
		listener(service, statusText+"\n")
	}
	return nil
}

func (logger *flatfileLogger) ProcessStatus(service string, status *client.SolveStatus) error {
	for _, v := range status.Vertexes { //e.g., [6/6] ADD app.py ./
		if strings.Index(v.Name, "[internal]") != 0 { //TODO HACK these are annoying
			if err := logger.Log(service, time.Now(), v.Name); err != nil {
				return errors.Errorf("Could not write to %s's logs: %s", service, err.Error())
			}
		}
	}

	for _, vs := range status.Statuses {
		if err := logger.logStatus(service, vs); err != nil {
			return errors.Errorf("Could not status to %s's logs: %s", service, err.Error())
		}
	}

	for _, log := range status.Logs {
		logMessage := []rune(string(log.Data))
		if err := logger.Log(service, log.Timestamp, strings.Trim(string(logMessage), "\r\n")); err != nil {
			return errors.Errorf("Could not write to %s's logs: %s", service, err.Error())
		}
	}
	return nil
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
