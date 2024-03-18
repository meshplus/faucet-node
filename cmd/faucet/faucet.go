package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/faucet/pkg/repo"
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
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show code version",
			Action: func(ctx *cli.Context) error {
				printVersion()
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
		os.Exit(-1)
	}
}

func printVersion() {
	fmt.Printf("%s version: %s-%s-%s\n", repo.AppName, repo.BuildVersion, repo.BuildBranch, repo.BuildCommit)
	fmt.Printf("App build date: %s\n", repo.BuildDate)
	fmt.Printf("System version: %s\n", repo.Platform)
	fmt.Printf("Golang version: %s\n", repo.GoVersion)
}
