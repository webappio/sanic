package shell

import (
	"io/ioutil"
	"strings"
	"text/template"
)

var BashShell = Shell {
	EnterArgs: func(sanicEnv string) (arguments []string) {
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

		err = tmpl.Execute(rcFile, TemplateData{Environment: sanicEnv})
		if err != nil {
			panic(err)
		}
		defer rcFile.Close()

		return []string{"--rcfile", rcFile.Name()}
	},

	ExecArgs: func(sanicEnv string, requestedCommand []string) (arguments []string) {
		var argumentPlaceholder strings.Builder //$0 $1 $2 ... $n
		argumentPlaceholder.WriteString("$0")
		for i := 1; i <= len(requestedCommand); i++ {
			argumentPlaceholder.WriteString(` "$`)
			argumentPlaceholder.WriteString(string(i))
			argumentPlaceholder.WriteRune('"')
		}

		for i, cmd := range requestedCommand {
			requestedCommand[i] = `'` + cmd + `'`
		}

		return append([]string{"-c", argumentPlaceholder.String()}, requestedCommand...)
	},

	ShellExecArgs: func(sanicEnv string, requestedCommand string) (arguments []string) {
		return []string{"-c", requestedCommand}
	},
}
