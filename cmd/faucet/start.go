package main

import (
	"faucet/app"
	"faucet/internal"
	"faucet/internal/loggers"
	"faucet/internal/repo"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/meshplus/bitxhub-kit/log"
	"github.com/urfave/cli"
)

var (
	startCMD = cli.Command{
		Name:   "start",
		Usage:  "Start a long-running daemon process",
		Action: start,
	}
)

func start(ctx *cli.Context) error {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return err
	}

	repo.SetPath(repoRoot)

	config, err := repo.UnmarshalConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("init config error: %s", err)
	}

	err = log.Initialize(
		log.WithReportCaller(config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(repoRoot, config.Log.Dir)),
		log.WithFileName(config.Log.Filename),
		log.WithMaxSize(2*1024*1024),
		log.WithMaxAge(24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("log initialize: %w", err)
	}
	// init loggers map for pier
	loggers.InitializeLogger(config)
	repo.SetPath(repoRoot)

	var server *app.Server
	var wg sync.WaitGroup
	wg.Add(1)
	handleShutdown(server, &wg)
	var client internal.Client
	err = client.Initialize(repoRoot)
	if err != nil {
		return err
	}
	server, _ = app.NewServer(&client)
	if err := server.Start(); err != nil {
		return err
	}
	wg.Wait()

	logger.Info("faucet exits")
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
