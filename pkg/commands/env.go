package commands

import (
	"fmt"
	"github.com/webappio/sanic/pkg/config"
	"github.com/webappio/sanic/pkg/provisioners"
	"github.com/webappio/sanic/pkg/shell"
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

func checkConfigAndEnv(configFile, envName string) error {
	projectName := filepath.Base(filepath.Dir(configFile))
	cfg, err := config.ReadFromPath(configFile)
	if err != nil {
		return err
	}
	env, ok := cfg.Environments[envName]
	if !ok {
		return fmt.Errorf("environment %s does not exist in project %s", env, projectName)
	}
	if err := provisioners.ValidateProvisionerConfig(env.ClusterProvisioner, env.ClusterProvisionerArgs); err != nil {
		return fmt.Errorf(
			"configuration file error: arguments provided to provisioner %s of type %s were invalid: %s",
			envName, env.ClusterProvisioner, err.Error())
	}
	return nil
}

func environmentCommandAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return newUsageError(c)
	}

	newSanicEnv := c.Args().First()

	createNewShell := false
	s, err := shell.Current()
	if err != nil {
		createNewShell = true
		configPath, err := findSanicConfig()
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		if configPath == "" {
			return cli.NewExitError(fmt.Sprintf("this command requires a %s file in your current dirctory or a parent directory.", SanicConfigName), 1)
		}
		err = checkConfigAndEnv(configPath, newSanicEnv)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		sanicRoot := filepath.Dir(configPath)
		s, err = shell.New(sanicRoot, configPath, newSanicEnv)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}
	err = checkConfigAndEnv(s.GetSanicConfig(), newSanicEnv)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	if c.NArg() == 1 {
		//sanic env dev
		if createNewShell {
			return cli.NewExitError(s.Enter(), 1)
		}
		err := s.ChangeEnvironment(newSanicEnv)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}
	//sanic env dev echo hello
	errorCode, err := s.Exec(c.Args()[1:])
	if err != nil {
		return cli.NewExitError(err.Error(), errorCode)
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
	if err != nil {
		return
	}

	var possibleEnvs []string

	for key := range configData.Environments {
		if strings.HasPrefix(key, requestedEnv) {
			possibleEnvs = append(possibleEnvs, key)
		}
	}
	if len(possibleEnvs) == 1 {
		print(possibleEnvs[0])
	}
	for env := range possibleEnvs {
		fmt.Println(env)
	}
}

var environmentCommand = cli.Command{
	Name:            "env",
	Usage:           "change to a specific (e.g., dev or production) environment named in the configuration",
	ArgsUsage:       "[environment name] (single command to execute...)",
	Action:          environmentCommandAction,
	BashComplete:    environmentCommandAutocomplete,
	SkipArgReorder:  true,
	SkipFlagParsing: true,
}
