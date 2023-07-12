package commands

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"strings"
	"syscall"
)

func enterCommandAction(cliContext *cli.Context) error {
	provisioner, err := getProvisioner()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if namespace == "" {
		namespace, err = getNamespaceFromEnv()
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}

	var args []string
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	args = append(args, "get", "pods", "-o", "jsonpath={.items[*].metadata.name}")
	cmd, err := provisioner.KubectlCommand(args...)
	if err != nil {
		return errors.Wrap(err, "error while getting kubernetes pods")
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
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

	cmd, err = provisioner.KubectlCommand("exec", "-it", filteredPodNames[0], "bash")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return cli.NewExitError(
		syscall.Exec(cmd.Path, cmd.Args, cmd.Env).Error(),
		1)
}

var namespace string

var enterCommand = cli.Command{
	Name:   "enter",
	Usage:  "sanic enter [pod unique name substring]",
	Action: enterCommandAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "n",
			Usage:       "specify namespace of pod to enter",
			Required:    false,
			Destination: &namespace,
		},
	},
}
