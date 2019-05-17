package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
	"strings"
	"time"
)

type plaintextInterfaceJob struct {
	totalJobLogs strings.Builder
	startTime    time.Time
}

type plaintextInterface struct {
	jobs map[string] *plaintextInterfaceJob
}

func NewPlaintextInterface() Interface {
	ret := plaintextInterface{}
	ret.jobs = make(map[string] *plaintextInterfaceJob)

	return &ret
}

func (iface plaintextInterface) FailJob(service string, err error) {
	if job, ok := iface.jobs[service]; ok {
		logs := job.totalJobLogs.String()
		if logs != "" {
			fmt.Printf("[%s] JOB FAILED: %s", service, err.Error())
			fmt.Printf("[%s] LOGS FOR FAILED SERVICE:\n", service)
			fmt.Print(logs)
			fmt.Printf("[%s] END FAILURE LOGS \n\n", service)
		} else {
			fmt.Printf("[%s] SERVICE FAILED TO BUILD WITHOUT LOGS.\n", service)
		}
	}
}

func (iface plaintextInterface) SucceedJob(service string) {
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

func (iface plaintextInterface) ProcessStatus(service string, status *client.SolveStatus) {
	if _, ok := iface.jobs[service]; !ok {
		iface.jobs[service] = &plaintextInterfaceJob{}
	}
	job := iface.jobs[service]
	logs := job.totalJobLogs
	for _, log := range status.Logs {
		if job.startTime.IsZero() {
			job.startTime = log.Timestamp
		}
		logs.WriteString(fmt.Sprintf("[t+%.2fs] ", log.Timestamp.Sub(job.startTime).Seconds()))
		logs.Write(log.Data)
	}
}