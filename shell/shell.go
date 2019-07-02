package shell

import (
	"github.com/pkg/errors"
	"os"
)

//Shell represents, broadly, the current shell environment we're in (by having executed sanic env)
//it keeps track of terminal state and allows executing commands.
type Shell interface {
	//Enter this specific shell.
	//If not already in a sanic env, this will execp a new shell
	//If already in a sanic env, this will change the current shell's variables
	//In both cases, only returns with an error, otherwise execution is terminated
	Enter() error

	//Execute the given command in this shell
	Exec(requestedCommand []string) (errorCode int, err error)

	//Execute the given command in "Shell mode", i.e., allowing spaces
	ShellExec(requestedCommand string, args []string) (errorCode int, err error)

	//If "sanic env dev", return "dev"
	GetSanicEnvironment() string

	//return absolute path to the current shell's sanic project's root
	GetSanicRoot() string

	//return absolute path to the current shell's sanic project's config
	GetSanicConfig() string

	//must already be in an environment, this changes the current shell's environment to a new one
	ChangeEnvironment(sanicEnvironment string) error
}

//Current gets the current shell (or an error if sanic env has not been used)
func Current() (Shell, error) {
	sanicRoot := os.Getenv("SANIC_ROOT")
	sanicConfig := os.Getenv("SANIC_CONFIG")
	sanicEnvironment := os.Getenv("SANIC_ENV")
	if sanicRoot == "" || sanicConfig == "" || sanicEnvironment == "" {
		return nil, errors.New("you must be in an environment to do this, see sanic env")
	}

	return New(sanicRoot, sanicConfig, sanicEnvironment)
}

//New creates a new sanic shell environment to execute commands in or to enter.
func New(sanicRoot, sanicConfig, sanicEnvironment string) (Shell, error) {
	shellPath := os.Getenv("BASH")
	if shellPath == "" {
		return nil, errors.New("only bash is supported. Try typing 'bash' into your terminal")
	}
	return &BashShell{
		Path:             shellPath,
		sanicRoot:        sanicRoot,
		sanicConfig:      sanicConfig,
		sanicEnvironment: sanicEnvironment,
	}, nil
}

func extraShellEnvironmentVars(shell Shell) []string {
	var env []string
	env = append(env, "SANIC_ENV="+shell.GetSanicEnvironment())
	env = append(env, "SANIC_ROOT="+shell.GetSanicRoot())
	env = append(env, "SANIC_CONFIG="+shell.GetSanicConfig())
	return env
}
