package main

import (
	"github.com/distributed-containers-inc/sanic/commands"
	"github.com/urfave/cli"
	"log"
	"os"
)

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

	app.Version = "1.0.0"
	app.Usage = "build & deploy kubernetes monorepos"

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
