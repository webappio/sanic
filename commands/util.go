package commands

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
)

const SanicConfigName = "sanic.yaml"

func getSanicEnv() string {
	//set in the env command
	return os.Getenv("SANIC_ENV")
}

func getSanicConfigPath() string {
	return os.Getenv("SANIC_CONFIG")
}

func findSanicConfig() (configPath string, err error) {
	currPath, err := filepath.Abs(".")
	if err != nil {
		return "", nil
	}
	for {
		if _, err := os.Stat(filepath.Join(currPath, SanicConfigName)); err == nil {
			return filepath.Abs(filepath.Join(currPath, SanicConfigName))
		}
		newPath, err := filepath.Abs(filepath.Join(currPath, ".."))
		if err != nil {
			return "", err
		}
		if newPath == currPath {
			return "", nil
		}
		currPath = newPath
	}
}

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
	return cli.NewExitError(err.Error(), exitCode);
}

