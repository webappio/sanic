package commands

import (
	"fmt"
	"github.com/urfave/cli"
)

func newUsageError(ctx *cli.Context) error {
	argsUsage := ctx.Command.ArgsUsage
	if argsUsage == "" {
		argsUsage = "[arguments ...]"
	}
	return cli.NewExitError(fmt.Sprintf(
		"Incorrect usage.\nCorrect usage: %s %s",
		ctx.Command.HelpName, argsUsage),
		1)
}

func wrapErrorWithExitCode(err error, exitCode int) *cli.ExitError {
	return cli.NewExitError(err.Error(), exitCode)
}
