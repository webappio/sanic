package commands

import (
	"fmt"
	"github.com/agnivade/levenshtein"
	"github.com/layer-devops/sanic/pkg/config"
	"github.com/layer-devops/sanic/pkg/shell"
	"github.com/urfave/cli"
	"sort"
	"strings"
)

func commandsMap(cfg *config.SanicConfig, env *config.Environment) map[string]config.Command {
	commands := make(map[string]config.Command)
	for _, globalCommand := range cfg.Commands {
		commands[globalCommand.Name] = globalCommand
	}
	for _, envCommand := range env.Commands {
		commands[envCommand.Name] = envCommand
	}
	return commands
}

func mostSimilarCommands(commands map[string]config.Command, requestedCommand string, num int) []string {
	var commandList []string
	for _, cmd := range commands {
		commandList = append(commandList, cmd.Name)
	}
	sort.Slice(commandList, func(i, j int) bool {
		distI := levenshtein.ComputeDistance(commandList[i], requestedCommand)
		distJ := levenshtein.ComputeDistance(commandList[j], requestedCommand)
		return distI < distJ
	})
	if len(commandList) <= num {
		return commandList
	}
	return commandList[:num]
}

func runCommandAction(cliCtx *cli.Context) error {
	s, err := shell.Current()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	cfg, err := config.Read()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	env, err := cfg.CurrentEnvironment(s)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	commands := commandsMap(&cfg, env)
	var commandName string
	if cliCtx.NArg() > 0 {
		commandName = cliCtx.Args().First()
	}

	if cmd, ok := commands[commandName]; ok {
		if cmd.Command == "" {
			return cli.NewExitError("Command "+commandName+" has an empty body in this environment.", 1)
		}
		code, err := s.ShellExec(cmd.Command, cliCtx.Args().Tail())
		if err == nil {
			return nil
		}
		return cli.NewExitError("", code)
	}

	return cli.NewExitError(
		fmt.Sprintf("Command %s was not found in environment %s. Did you mean one of [%s]?",
			commandName,
			s.GetSanicEnvironment(),
			strings.Join(mostSimilarCommands(commands, commandName, 6), "|"),
		), 1)

}

var runCommand = cli.Command{
	Name:            "run",
	Usage:           "run a configured script in the configuration",
	Action:          runCommandAction,
	SkipArgReorder:  true,
	SkipFlagParsing: true,
}
