package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/axiomesh/faucet/pkg/repo"
)

var configCMD = &cli.Command{
	Name:  "config",
	Usage: "The config manage commands",
	Subcommands: []*cli.Command{
		{
			Name:   "generate",
			Usage:  "Generate default config and node private key(if not exist)",
			Action: generate,
		},
	},
}

func generate(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}
	if fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("faucet repo already exists")
		return nil
	}

	if !fileutil.Exist(p) {
		err = os.MkdirAll(p, 0755)
		if err != nil {
			return err
		}
	}

	c := repo.DefaultConfig()
	r := &repo.Repo{
		RepoRoot: p,
		Config:   c,
	}
	if err := r.Flush(); err != nil {
		return err
	}
	return nil
}

func getRootPath(ctx *cli.Context) (string, error) {
	p := ctx.String("repo")

	var err error
	if p == "" {
		p, err = repo.LoadRepoRootFromEnv(p)
		if err != nil {
			return "", err
		}
	}
	return p, nil
}
