package shell

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

//BashShell returns a new "wrapped" bash-based shell, which has the correct environment variables and prompt
var BashShell = Shell{
	EnterArgs: func(sanicEnv, configPath string) (arguments []string) {
		tmpl, err := template.New("rcfile").Parse(
			`
source ~/.bashrc

if [ -z "${OLD_PROMPT_COMMAND+x}" ]; then
  OLD_PROMPT_COMMAND="$PROMPT_COMMAND"
fi
if [ -z "${OLD_PS1+x}" ]; then
  OLD_PS1="$PS1"
fi

# delete this file when this shell exits
trap 'rm {{.RCFile}}' INT TERM EXIT

export SANIC_ENV='{{.Environment}}'
export SANIC_ROOT='{{.Root}}'
export SANIC_CONFIG='{{.ConfigPath}}'

# 1. save exit status of last command (e.g., in case they change prompt color)
# 2. save old PS1 (e.g., in case they don't set PS1, we don't want it to keep appending [dev]
# 3. run their prompt command (if any)
# 4. append [dev] in front
PROMPT_COMMAND='status=$?; PS1="$OLD_PS1"; ( exit $status; ); '"$OLD_PROMPT_COMMAND"'; PS1="[$SANIC_ENV] $PS1"; '
`)

		type TemplateData struct {
			Environment string
			RCFile      string
			Root        string
			ConfigPath  string
		}

		if err != nil {
			panic(err)
		}

		rcFile, err := ioutil.TempFile("", "sanic-rcfile-*.bash")

		if err != nil {
			panic(err)
		}

		err = tmpl.Execute(rcFile, TemplateData{
			Environment: sanicEnv,
			RCFile:      rcFile.Name(),
			Root:        filepath.Dir(configPath),
			ConfigPath:  configPath,
		})
		if err != nil {
			panic(err)
		}
		defer rcFile.Close()

		return []string{"--rcfile", rcFile.Name()}
	},

	ExecArgs: func(sanicEnv string, requestedCommand []string) (arguments []string) {
		var argumentPlaceholder strings.Builder //$0 $1 $2 ... $n
		for i := 0; i < len(requestedCommand); i++ {
			argumentPlaceholder.WriteString(` "$`)
			argumentPlaceholder.WriteString(strconv.Itoa(i))
			argumentPlaceholder.WriteRune('"')
		}

		return append([]string{"-c", argumentPlaceholder.String()}, requestedCommand...)
	},

	ShellExecArgs: func(sanicEnv string, requestedCommand string) (arguments []string) {
		return []string{"-c", requestedCommand}
	},
}
