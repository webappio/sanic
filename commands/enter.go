package commands

import (
	"bytes"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func enterCommandAction(cliContext *cli.Context) error {
	if cliContext.NArg() != 1 {
		return newUsageError(cliContext)
	}

	kubeExecutableLocation, err := exec.LookPath("kubectl")
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not find kubectl, is it installed? %s", err.Error()), 1)
	}

	provisioner, err := getProvisioner()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return cli.NewExitError("the kubernetes configuration doesn't exist yet, use sanic deploy first", 1)
	}
	cmd := exec.Command("kubectl", "get", "pods", "-o", "jsonpath={.items[*].metadata.name}")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeConfigLocation)
	err = cmd.Run()
	if err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		return cli.NewExitError(fmt.Sprintf("could not get pods: %s", err.Error()), 1)
	}
	if len(strings.TrimSpace(stdout.String())) == 0 {
		return cli.NewExitError("there are no pods running in the current namespace.", 1)
	}

	podNames := strings.Split(strings.TrimSpace(stdout.String()), " ")
	var filteredPodNames []string
	filterString := cliContext.Args().First()
	for _, podName := range podNames {
		if strings.Contains(podName, filterString) {
			filteredPodNames = append(filteredPodNames, podName)
		}
	}
	if len(filteredPodNames) == 0 {
		return cli.NewExitError(
			fmt.Sprintf("there are no pods that match %s in the current namespace.", filterString),
			1)
	}
	if len(filteredPodNames) > 1 {
		return cli.NewExitError(
			fmt.Sprintf("there are multiple pods that match %s in the current namespace: %s", filterString, strings.Join(filteredPodNames, ", ")),
			1)
	}

	env, err := getKubectlEnvironment()

	return cli.NewExitError(
		syscall.Exec(kubeExecutableLocation, []string{"kubectl", "exec", "-it", filteredPodNames[0], "bash"}, env).Error(),
		1)
}

var enterCommand = cli.Command{
	Name:   "enter",
	Usage:  "sanic enter [pod unique name substring]",
	Action: enterCommandAction,
}
