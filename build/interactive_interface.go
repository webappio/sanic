package build

import (
	"fmt"
	"github.com/gdamore/tcell"
	"sort"
	"strings"
	"sync"
	"time"
)

type interactiveInterfaceJob struct {
	lastNonemptyLog     string
	lastNonemptyLogTime time.Time
	status              string
	service             string
}

type interactiveInterface struct {
	jobs            map[string]*interactiveInterfaceJob
	mutex           sync.Mutex
	screen          tcell.Screen
	screenStyle     tcell.Style
	running         bool
	cancelListeners []func()
}

//NewInteractiveInterface creates and initializes a new tcell screen and event loop for use as an Interface
func NewInteractiveInterface() (Interface, error) {
	iface := &interactiveInterface{
		screenStyle: tcell.StyleDefault,
		jobs:        make(map[string]*interactiveInterfaceJob),
		running:     true,
	}

	tcell.SetEncodingFallback(tcell.EncodingFallbackFail)
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err = screen.Init(); err != nil {
		return nil, err
	}
	screen.Clear()

	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			switch typedEvent := ev.(type) {
			case *tcell.EventResize:
				iface.redrawScreen()
				screen.Sync()
			case *tcell.EventKey:
				switch typedEvent.Key() {
				case tcell.KeyCtrlC, tcell.KeyEsc, tcell.KeyExit:
					for _, cancel := range iface.cancelListeners {
						cancel()
					}
					return
				}
			}
		}
	}()

	go func() {
		for iface.running {
			iface.redrawScreen()
			time.Sleep(time.Millisecond * 150)
		}
	}()

	iface.screen = screen

	return iface, nil
}

func (iface interactiveInterface) redrawScreen() {
	width, height := iface.screen.Size()

	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	var succeededJobs []*interactiveInterfaceJob
	var failedJobs []*interactiveInterfaceJob
	var currJobs []*interactiveInterfaceJob

	for _, job := range iface.jobs {
		switch job.status {
		case "succeeded":
			succeededJobs = append(succeededJobs, job)
		case "failed":
			failedJobs = append(failedJobs, job)
		default:
			currJobs = append(currJobs, job)
		}
	}

	sortJobs := func(jobs []*interactiveInterfaceJob) {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].service < jobs[j].service
		})
	}
	sortJobs(succeededJobs)
	sortJobs(failedJobs)
	sortJobs(currJobs)

	displayAndTruncateString := func(y int, s string, style tcell.Style) {
		for i := 0; i < width && i < len(s); i++ {
			iface.screen.SetContent(i, y, []rune(s)[i], []rune{}, style)
		}
		for i := len(s); i < width; i++ {
			iface.screen.SetContent(i, y, ' ', []rune{}, style)
		}
	}

	currRenderLine := 0

	failureStyle := iface.screenStyle.Foreground(tcell.NewRGBColor(190, 0, 0))
	for _, failedJob := range failedJobs {
		if currRenderLine >= height-1 {
			break
		}
		displayAndTruncateString(currRenderLine, "[failed] "+failedJob.service, failureStyle)
		displayAndTruncateString(currRenderLine+1, failedJob.lastNonemptyLog, iface.screenStyle)
		currRenderLine += 2
	}

	currStyle := iface.screenStyle.Foreground(tcell.NewRGBColor(190, 190, 0))
	for _, currJob := range currJobs {
		if currRenderLine >= height-1 {
			break
		}
		displayAndTruncateString(currRenderLine, "[building] "+currJob.service, currStyle)
		displayAndTruncateString(currRenderLine+1, currJob.lastNonemptyLog, iface.screenStyle)
		currRenderLine += 2
	}

	succeededStyle := iface.screenStyle.Foreground(tcell.NewRGBColor(0, 190, 0))
	for _, succeededJob := range succeededJobs {
		if currRenderLine >= height-1 {
			break
		}
		displayAndTruncateString(currRenderLine, "[complete] "+succeededJob.service, succeededStyle)
		displayAndTruncateString(currRenderLine+1, succeededJob.lastNonemptyLog, iface.screenStyle)
		currRenderLine += 2
	}

	iface.screen.Show()
}

func (iface *interactiveInterface) Close() {
	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	iface.running = false
	iface.screen.Fini()
	var serviceNames []string
	anyJobsFailed := false
	allJobsFailed := true
	for jobName, job := range iface.jobs {
		serviceNames = append(serviceNames, jobName)
		if job.status == "succeeded" {
			allJobsFailed = false
		} else {
			anyJobsFailed = true
		}
	}

	if allJobsFailed {
		fmt.Printf("Failed to build some of the following jobs: %s\nSee the logs folder for details.\n", strings.Join(serviceNames, ", "))
	} else if !anyJobsFailed {
		fmt.Printf("Successfully built: %s\n", strings.Join(serviceNames, ", "))
	} //otherwise, build was cancelled

}

func (iface *interactiveInterface) StartJob(service string) {
	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	iface.jobs[service] = &interactiveInterfaceJob{service: service}
}

func (iface *interactiveInterface) FailJob(service string, err error) {
	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	if job, ok := iface.jobs[service]; ok {
		job.status = "failed"
	}
}

func (iface *interactiveInterface) SucceedJob(service string) {
	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	if job, ok := iface.jobs[service]; ok {
		job.status = "succeeded"
	}
}

func (iface *interactiveInterface) ProcessLog(service, logLine string) {
	iface.mutex.Lock()
	defer iface.mutex.Unlock()

	job, ok := iface.jobs[service]
	if !ok {
		panic("Could not find service: " + service)
	}
	logLine = strings.TrimSpace(logLine)
	if logLine != "" {
		job.lastNonemptyLog = logLine
		//notice: server time might drift, so we use local time
		job.lastNonemptyLogTime = time.Now()
	}
}

func (iface *interactiveInterface) AddCancelListener(cancelFunc func()) {
	iface.cancelListeners = append(iface.cancelListeners, cancelFunc)
}
