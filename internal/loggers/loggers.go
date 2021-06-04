package loggers

import (
	repo "faucet/internal/repo"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/sirupsen/logrus"
)

const (
	ApiServer = "api_server"
)

var w *loggerWrapper

type loggerWrapper struct {
	loggers map[string]*logrus.Entry
}

func InitializeLogger(config *repo.Config) {
	m := make(map[string]*logrus.Entry)
	m[ApiServer] = log.NewWithModule(ApiServer)
	m[ApiServer].Logger.SetLevel(log.ParseLevel(config.Log.Module.ApiServer))

	w = &loggerWrapper{loggers: m}
}

func Logger(name string) logrus.FieldLogger {
	return w.loggers[name]
}
