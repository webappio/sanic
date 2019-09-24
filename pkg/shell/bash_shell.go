package shell

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

//BashShell represents a bash environment which maintains sanic's environment variables
type BashShell struct {
	Path             string //e.g., /bin/bash
	sanicRoot        string //absolute path to the directory which contains the sanic config
	sanicConfig      string //absolute path to the sanic configuration file
	sanicEnvironment string //name of the environment we are in
}

func (shell *BashShell) writeEnvFile(envFile *os.File) error {
	for _, env := range extraShellEnvironmentVars(shell) {
		_, err := envFile.WriteString(env)
		if err != nil {
			return err
		}
		_, err = envFile.WriteString("\n")
		if err != nil {
			return err
		}
	}
	return nil
}

//Enter : execvp the current process into a new sanic shell. Note: Does not return.
func (shell *BashShell) Enter() error {
	tmpl, err := template.New("rcfile").Parse(
		`
source ~/.bashrc

if [ -z "${OLD_PROMPT_COMMAND+x}" ]; then
  OLD_PROMPT_COMMAND="${PROMPT_COMMAND:-true}"
  # trim leading / trailing whitespace
  OLD_PROMPT_COMMAND=$(echo -e "${OLD_PROMPT_COMMAND}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
fi
if [ -z "${OLD_PS1+x}" ]; then
  OLD_PS1="$PS1"
fi

export SANIC_BASH_ENV_FILE='{{.EnvFile}}'
set -a # export the loaded variables
source "${SANIC_BASH_ENV_FILE}"
set +a

# delete this file when this shell exits
trap 'rm {{.RCFile}}; rm {{.EnvFile}}' EXIT

# reload any environment changes before every command
trap 'set -a; source "${SANIC_BASH_ENV_FILE}"; set +a' DEBUG

# 1. save exit status of last command (e.g., in case they change prompt color)
# 2. save old PS1 (e.g., in case they don't set PS1, we don't want it to keep appending [dev]
# 3. run their prompt command (if any)
# 4. append [dev] in front
PROMPT_COMMAND='status=$?; PS1="$OLD_PS1"; ( exit $status; ); '"${OLD_PROMPT_COMMAND}"'; PS1="[$(basename $SANIC_ROOT) $SANIC_ENV] $PS1"'
`)

	type TemplateData struct {
		RCFile  string
		EnvFile string
	}

	if err != nil {
		return err
	}

	rcFile, err := ioutil.TempFile("", "sanic-rcfile-*.bash")
	defer rcFile.Close()

	if err != nil {
		return err
	}

	sanicEnvFile, err := ioutil.TempFile("", "sanic-env-*.env")
	defer sanicEnvFile.Close()
	if err != nil {
		return err
	}
	shell.writeEnvFile(sanicEnvFile)

	err = tmpl.Execute(rcFile, TemplateData{
		RCFile:  rcFile.Name(),
		EnvFile: sanicEnvFile.Name(),
	})
	if err != nil {
		return err
	}

	return syscall.Exec(
		shell.Path,
		[]string{shell.Path, "--rcfile", rcFile.Name()},
		os.Environ())
}

//Exec : execute the given command without shell interpolation
func (shell *BashShell) Exec(requestedCommand []string) (errorCode int, err error) {
	var argumentPlaceholder strings.Builder //$0 $1 $2 ... $n
	for i := 0; i < len(requestedCommand); i++ {
		argumentPlaceholder.WriteString(` "$`)
		argumentPlaceholder.WriteString(strconv.Itoa(i))
		argumentPlaceholder.WriteRune('"')
	}

	cmd := exec.Command(shell.Path, append([]string{"-c", argumentPlaceholder.String()}, requestedCommand...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), extraShellEnvironmentVars(shell)...)
	err = cmd.Start()
	if err != nil {
		errorCode = 1
		return
	}
	err = cmd.Wait()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				errorCode = status.ExitStatus()
				return
			}
		}
		errorCode = 1
		return
	}
	errorCode = 0
	return
}

//ShellExec : execute the given shell command (i.e., including spaces) in the given environment
func (shell *BashShell) ShellExec(requestedCommand string, args []string) (errorCode int, err error) {
	cmd := exec.Command(shell.Path, append([]string{"-c", requestedCommand, "dummy"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), extraShellEnvironmentVars(shell)...)
	cmd.Dir = shell.sanicRoot
	err = cmd.Start()
	if err != nil {
		errorCode = 1
		return
	}
	err = cmd.Wait()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				errorCode = status.ExitStatus()
				return
			}
		}
		errorCode = 1
		return
	}
	errorCode = 0
	return
}

//GetSanicEnvironment returns the current environment (e.g., "sanic env dev" -> dev)
func (shell *BashShell) GetSanicEnvironment() string {
	return shell.sanicEnvironment
}

//GetSanicConfig returns the current path to the sanic configuration file
func (shell *BashShell) GetSanicConfig() string {
	return shell.sanicConfig
}

//GetSanicRoot returns the current path to the project root directory
func (shell *BashShell) GetSanicRoot() string {
	return shell.sanicRoot
}

//ChangeEnvironment changes the currently entered shell's sanic environment to a new one
func (shell *BashShell) ChangeEnvironment(sanicEnvironment string) error {
	envFile := os.Getenv("SANIC_BASH_ENV_FILE")
	if envFile == "" {
		return errors.New("the environment file variable is not set")
	}
	f, err := os.OpenFile(envFile, os.O_WRONLY|os.O_CREATE, 0)
	defer f.Close()
	if err != nil {
		return err
	}
	oldEnv := shell.sanicEnvironment
	shell.sanicEnvironment = sanicEnvironment
	err = shell.writeEnvFile(f)
	if err != nil {
		shell.sanicEnvironment = oldEnv
	}
	return err
}
