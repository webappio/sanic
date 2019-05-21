package build

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

type plaintextInterfaceJob struct {
	totalJobLogs strings.Builder
	startTime    time.Time
}

type plaintextInterface struct {
	jobs            map[string]*plaintextInterfaceJob
	cancelListeners []func()
}

func addSignalCanceller(iface *plaintextInterface) {
	go func() {
		signals := make(chan os.Signal, 2048)
		signal.Notify(signals, os.Interrupt)
		for retries := 0; retries < 3; retries++ {
			<-signals
			for _, cancel := range iface.cancelListeners {
				cancel()
			}
		}
		fmt.Fprintln(os.Stderr, "Forcing shutdown.")
		os.Exit(1)
	}()
}

//NewPlaintextInterface initializes a new plaintext (e.g., no advanced terminal required) Interface
func NewPlaintextInterface() Interface {
	iface := plaintextInterface{}
	iface.jobs = make(map[string]*plaintextInterfaceJob)
	addSignalCanceller(&iface)

	return &iface
}

func (iface *plaintextInterface) Close() {
	//do nothing
}

func (iface *plaintextInterface) StartJob(service string) {
	iface.jobs[service] = &plaintextInterfaceJob{}
}

func (iface *plaintextInterface) FailJob(service string, err error) {
	if job, ok := iface.jobs[service]; ok {
		logs := job.totalJobLogs.String()
		if err == context.Canceled {
			//do nothing: job was cancelled
		} else if logs != "" {
			fmt.Printf("[%s] JOB FAILED: %s\n", service, err.Error())
			fmt.Printf("[%s] LOGS FOR FAILED SERVICE:\n", service)
			fmt.Print(logs)
			fmt.Printf("[%s] END FAILURE LOGS \n\n", service)
		} else {
			fmt.Printf("[%s] SERVICE FAILED TO BUILD WITHOUT LOGS.\n", service)
		}
	}
}

func (iface *plaintextInterface) SucceedJob(service string) {
	if job, ok := iface.jobs[service]; ok {
		logs := job.totalJobLogs.String()
		if logs != "" {
			fmt.Printf("[%s] Logs for successfully built service:\n", service)
			fmt.Print(logs)
			fmt.Printf("[%s] End of logs.\n\n", service)
		} else {
			fmt.Printf("[%s] Service built.\n", service)
		}
	}
}

func (iface *plaintextInterface) ProcessLog(service, logLine string) {
	job := iface.jobs[service]
	logs := &job.totalJobLogs
	if job.startTime.IsZero() {
		job.startTime = time.Now()
	}
	logs.WriteString(fmt.Sprintf("[t+%.2fs] ", time.Now().Sub(job.startTime).Seconds()))
	logs.WriteString(logLine)
}

func (iface *plaintextInterface) AddCancelListener(cancelFunc func()) {
	iface.cancelListeners = append(iface.cancelListeners, cancelFunc)
}
