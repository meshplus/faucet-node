package main

import (
	"os"
	"time"

	"github.com/axiomesh/axiom-kit/log"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var logger = log.NewWithModule("cmd")

func main() {
	app := cli.NewApp()
	app.Name = "Faucet"
	app.Usage = "Get the axm node"
	app.Compiled = time.Now()

	// global flags
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "repo",
			Usage: "Work path",
		},
	}

	app.Commands = []*cli.Command{
		configCMD,
		startCMD,
	}

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
		os.Exit(-1)
	}
}
