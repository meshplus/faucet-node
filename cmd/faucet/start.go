package main

import (
	"context"
	"faucet/app"
	"faucet/internal"
	"faucet/pkg/loggers"
	"faucet/pkg/repo"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/axiomesh/axiom-kit/fileutil"
	"github.com/common-nighthawk/go-figure"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	startCMD = &cli.Command{
		Name:   "start",
		Usage:  "Start a long-running daemon process",
		Action: start,
	}
)

func start(ctx *cli.Context) error {
	p, err := getRootPath(ctx)
	if err != nil {
		return err
	}

	if !fileutil.Exist(filepath.Join(p, repo.CfgFileName)) {
		fmt.Println("faucet config not found")
		return err
	}

	repo, err := repo.Load(p)
	if err != nil {
		return err
	}
	appCtx, cancel := context.WithCancel(ctx.Context)
	if err := loggers.Initialize(appCtx, repo, true); err != nil {
		cancel()
		return err
	}
	defer cancel()

	log := loggers.Logger(loggers.Global)

	var server *app.Server
	var wg sync.WaitGroup
	wg.Add(1)
	handleShutdown(server, &wg)
	var client internal.Client
	err = client.Initialize(repo.Config, p)
	if err != nil {
		log.Error(err)
		return err
	}
	server, _ = app.NewServer(&client, repo.Config)
	if err := server.Start(); err != nil {
		log.Error(err)
		return err
	}

	printLogo(log)
	wg.Wait()
	return nil
}

func handleShutdown(server *app.Server, wg *sync.WaitGroup) {
	var stop = make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(stop, syscall.SIGINT)

	go func() {
		<-stop
		fmt.Println("received interrupt signal, shutting down...")
		if err := server.Stop(); err != nil {
			logger.Error("faucet stop: ", err)
		}

		wg.Done()
		os.Exit(0)
	}()
}

func printLogo(log logrus.FieldLogger) {
	fig := figure.NewFigure("Faucet", "slant", true)
	log.WithField("__format_only_write_msg_without_formatter", nil).Infof(`
=========================================================================================
%s
=========================================================================================
`, fig.String())

}
