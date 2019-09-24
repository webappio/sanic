package commands

import "github.com/urfave/cli"

//Commands is the default list of commands for sanic (e.g., env, build, run, ...)
var Commands = []cli.Command{
	buildCommand,
	deployCommand,
	enterCommand,
	environmentCommand,
	kubectlCommand,
	runCommand,
}
