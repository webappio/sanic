package build

import (
	"github.com/moby/buildkit/client"
)

type Interface interface {
	FailJob(service string, err error)
	SucceedJob(service string)
	ProcessStatus(service string, status *client.SolveStatus)
}
