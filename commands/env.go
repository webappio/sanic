package commands

import (
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"text/template"
)

func getShell() (dir, file string) {
	return filepath.Split(os.Getenv("SHELL"))
}

func parseBashFlags(environment string) []string {
	tmpl, err := template.New("rcfile").Parse(
		`
source ~/.bashrc

if [ -z "${OLD_PROMPT_COMMAND+x}" ]; then
  OLD_PROMPT_COMMAND="$PROMPT_COMMAND"
  OLD_PS1="$PS1"
  export SANIC_ENV='{{.Environment}}'
fi
PROMPT_COMMAND='PS1="$OLD_PS1"; '"$OLD_PROMPT_COMMAND; "'export PS1="[$SANIC_ENV] $PS1"; '
`)

	type TemplateData struct {
		Environment string
	}

	if err != nil {
		panic(err)
	}

	rcFile, err := ioutil.TempFile("", "sanic-rcfile-*.bash")

	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(rcFile, TemplateData{Environment: environment})
	if err != nil {
		panic(err)
	}
	defer rcFile.Close()

	return []string{"--rcfile", rcFile.Name()}
}

func environmentCommandAction(c *cli.Context) error {
	if len(c.Args()) != 1 {
		return newUsageError(c)
	}

	environment := c.Args().First()

	shellDir, currShell := getShell()
	shellPath := filepath.Join(shellDir, currShell)

	if currShell != "bash" {
		return cli.NewExitError("only bash is supported for the env command", 1)
	}

	return wrapErrorWithExitCode(
		syscall.Exec(shellPath, append([]string{shellPath}, parseBashFlags(environment)...), os.Environ()),
		1)
}

var EnvironmentCommand = cli.Command{
	Name:      "env",
	Usage:     "change to a specific (e.g., dev or production) environment named in the configuration",
	ArgsUsage: "[environment name]",
	Action:    environmentCommandAction,
}
