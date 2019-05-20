package commands

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strings"
)

//SanicConfigName is the name of the configuration file to read.
//It also functions as denoting the root directory of the monorepo.
//sanic env searches for this to allow you to enter environments easily.
const SanicConfigName = "sanic.yaml"

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

func environmentCommandAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return newUsageError(c)
	}

	sanicEnv := c.Args().First()
	configPath, err := findSanicConfig()
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}
	if configPath == "" {
		return cli.NewExitError(fmt.Sprintf("this command requires a %s file in your current dirctory or a parent directory.", SanicConfigName), 1)
	}
	sanicRoot := filepath.Base(configPath)
	s, err := shell.New(sanicRoot, configPath, sanicEnv)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	cfg, err := config.ReadFromPath(configPath)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}
	_, err = cfg.CurrentEnvironment(s)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	if c.NArg() == 1 {
		//sanic env dev
		//if this returns, there's been an error (we execp here)
		return wrapErrorWithExitCode(s.Enter(), 1)
	}
	//sanic env dev echo hello
	errorCode, err := s.Exec(c.Args()[1:])
	if err != nil {
		return wrapErrorWithExitCode(err, errorCode)
	}
	return nil
}

func environmentCommandAutocomplete(c *cli.Context) {
	if c.NArg() > 1 {
		return
	}
	var requestedEnv = ""
	if c.NArg() == 1 {
		requestedEnv = c.Args().First()
	}
	configPath, err := findSanicConfig()
	if err != nil {
		return
	}
	configData, err := config.ReadFromPath(configPath)
	if err != nil || configData == nil {
		return
	}

	var possibleEnvs = []string{}

	for key := range configData.Environments {
		if strings.HasPrefix(key, requestedEnv) {
			possibleEnvs = append(possibleEnvs, key)
		}
	}
	if len(possibleEnvs) == 1 {
		print(possibleEnvs[0])
	}
	for env := range possibleEnvs {
		println(env)
	}
}

var environmentCommand = cli.Command{
	Name:         "env",
	Usage:        "change to a specific (e.g., dev or production) environment named in the configuration",
	ArgsUsage:    "[environment name] (single command to execute...)",
	Action:       environmentCommandAction,
	BashComplete: environmentCommandAutocomplete,
}
