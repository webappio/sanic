package build

import (
	"fmt"
	"github.com/moby/buildkit/client"
)

type Interface interface {
	FailJob(service string, err error)
	ProcessStatus(service string, status *client.SolveStatus)
}

type PlaintextInterface struct {

}

func (PlaintextInterface) FailJob(service string, e error) {
	fmt.Printf("[%s] Failed: %s\n", service, e.Error())
}

func (PlaintextInterface) ProcessStatus(service string, status *client.SolveStatus) {
	for _, log := range status.Logs {
		fmt.Printf("[%s] %s %s\n", service, log.Timestamp, log.Data)
	}
}

type CursesInterface struct {

}

func (CursesInterface) FailJob(service string, err error) {
	panic("implement me")
}

func (CursesInterface) ProcessStatus(service string, status *client.SolveStatus) {
	panic("implement me")
}
