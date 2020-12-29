package main

import (
	"github.com/layer-devops/sanic/pkg/commands"
	"github.com/urfave/cli"
	"log"
	"os"
)

//version is the version of this app.
//Use --ldflags '-X main.version=(...)' with go build to update.
var version = "master"

func main() {

	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "Location of the configuration file.",
		},
	}

	app.Commands = commands.Commands

	app.EnableBashCompletion = true

	app.Version = version
	app.Usage = "build & deploy kubernetes monorepos"

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
