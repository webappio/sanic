package shell

import (
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func environment() map[string]string {
	env := os.Environ()
	result := make(map[string]string, len(env))
	for _, item := range env {
		split := strings.SplitN(item, "=", 2)
		result[split[0]] = split[1]
	}
	return result
}

func environmentToList(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

//EnvironmentVariables returns the current environment variables, appended with Sanic's required variables
func EnvironmentVariables(sanicEnv, configPath string) []string {
	envVars := environment()
	envVars["SANIC_ENV"] = sanicEnv
	envVars["SANIC_CONFIG"] = configPath
	envVars["SANIC_ROOT"] = filepath.Dir(configPath)

	return environmentToList(envVars)
}

//Shell is a way to execute commands in, and interact with, Sanic.  It provides methods to:
//1. enter the shell (e.g., "sanic env dev')
//2. execute something directly in the shell (e.g., "sanic env dev echo hello world")
//3. execute something as a shell script in the shell (e.g., "sanic run echo_hello")
type Shell struct {
	//Return the arguments to run an interactive version of a specific shell
	//e.g., if executable is /bin/bash, this might return --rcfile /tmp/blah.bash
	EnterArgs func(sanicEnv string) (arguments []string)

	//Return the arguments to run the given shell contents in a
	// "one-shot" version of this specific shell, preserving arguments
	//e.g., bash -c '"$0" "$1" "$2"' 'echo' 'hello' 'world' if given echo hello world
	ExecArgs func(sanicEnv string, requestedCommand []string) (arguments []string)

	//Return the flags to run the given command naively in "shell" model
	//e.g., bash -c 'echo hello world' given 'echo hello world' as a single string
	ShellExecArgs func(sanicEnv string, requestedCommand string) (arguments []string)
}

func getShell() (shellPath string, shell *Shell, err error) {
	shellPath = os.Getenv("SHELL")
	shellName := filepath.Base(shellPath)
	if shellName != "bash" {
		err = errors.New("only bash is supported for this operation")
		return
	}
	shell = &BashShell
	return
}

//Enter : execvp the current process into a new sanic shell. Note: Does not return.
func Enter(sanicEnv, configPath string) error {
	shellPath, shell, err := getShell()
	if err != nil {
		return err
	}
	argv := []string{shellPath}
	argv = append(argv, shell.EnterArgs(sanicEnv)...)
	return syscall.Exec(
		shellPath,
		argv,
		EnvironmentVariables(sanicEnv, configPath))
}

//EnterExec : execute the given command in the given environment
func EnterExec(sanicEnv, configPath string, requestedCommand []string) (errorCode int, err error) {
	shellPath, shell, err := getShell()
	if err != nil {
		errorCode = 1
		return
	}
	println(strings.Join(shell.ExecArgs(sanicEnv, requestedCommand), ", "))
	cmd := exec.Command(shellPath, shell.ExecArgs(sanicEnv, requestedCommand)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = EnvironmentVariables(sanicEnv, configPath)
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

//Exec : execute the given (shell, given as string) command in the given environment
func Exec(sanicEnv, configPath string, requestedCommand string) (errorCode int, err error) {
	shellPath, shell, err := getShell()
	if err != nil {
		errorCode = 1
		return
	}
	cmd := exec.Command(shellPath, shell.ShellExecArgs(sanicEnv, requestedCommand)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = EnvironmentVariables(sanicEnv, configPath)
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
