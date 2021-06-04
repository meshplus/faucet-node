package loggers

import (
	repo "faucet/internal/repo"
	"testing"
)

func TestLogger(t *testing.T) {
	config := &repo.Config{
		Log: repo.Log{
			Dir:          "logs",
			Filename:     "faucet.log",
			ReportCaller: false,
			Level:        "info",
			Module: repo.LogModule{

				ApiServer: "info",
			},
		},
	}
	InitializeLogger(config)
	Logger(ApiServer).Info("api_server")
}
