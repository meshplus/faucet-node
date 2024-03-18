package loggers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/axiomesh/axiom-kit/log"
	"github.com/axiomesh/faucet/pkg/repo"
)

const (
	ApiServer = "api_server"
	Global    = "global"
)

var w = &LoggerWrapper{
	loggers: map[string]*logrus.Entry{
		ApiServer: log.NewWithModule(ApiServer),
	},
}

type LoggerWrapper struct {
	loggers map[string]*logrus.Entry
}

func Initialize(ctx context.Context, rep *repo.Repo, persist bool) error {
	config := rep.Config
	err := log.Initialize(
		log.WithCtx(ctx),
		log.WithEnableCompress(config.Log.EnableCompress),
		log.WithReportCaller(config.Log.ReportCaller),
		log.WithEnableColor(config.Log.EnableColor),
		log.WithDisableTimestamp(config.Log.DisableTimestamp),
		log.WithPersist(persist),
		log.WithFilePath(filepath.Join(rep.RepoRoot, repo.LogsDirName)),
		log.WithFileName(config.Log.Filename),
		log.WithMaxAge(int(config.Log.MaxAge)),
		log.WithMaxSize(int(config.Log.MaxSize)),
		log.WithRotationTime(config.Log.RotationTime.ToDuration()),
	)
	if err != nil {
		return fmt.Errorf("log initialize: %w", err)
	}

	m := make(map[string]*logrus.Entry)
	m[ApiServer] = log.NewWithModule(ApiServer)
	m[ApiServer].Logger.SetLevel(log.ParseLevel(config.Log.Module.ApiServer))
	m[Global] = log.NewWithModule(Global)
	m[Global].Logger.SetLevel(log.ParseLevel(config.Log.Module.Global))

	w = &LoggerWrapper{loggers: m}
	return nil
}

func Logger(name string) logrus.FieldLogger {
	return w.loggers[name]
}
