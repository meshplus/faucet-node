package main

import (
	"bufio"
	"faucet/internal/repo"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/urfave/cli"
)

var initCMD = cli.Command{
	Name:  "init",
	Usage: "Initialize faucet local configuration",
	Action: func(ctx *cli.Context) error {
		repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
		if err != nil {
			return err
		}

		if fileutil.Exist(filepath.Join(repoRoot, repo.ConfigName)) {
			fmt.Println("faucet configuration file already exists")
			fmt.Println("reinitializing would overwrite your configuration, Y/N?")
			input := bufio.NewScanner(os.Stdin)
			input.Scan()
			if input.Text() == "Y" || input.Text() == "y" {
				return repo.Initialize(repoRoot)
			}
			return nil
		}

		return repo.Initialize(repoRoot)
	},
}
