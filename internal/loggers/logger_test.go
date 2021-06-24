package loggers

import (
	repo "faucet/internal/repo"
	"fmt"
	"testing"

	"github.com/meshplus/bitxhub-kit/types"
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

func TestLogger2(t *testing.T) {
	if add := types.NewAddressByStr("8c0B7b40E03e4C76Q"); add == nil {
		fmt.Println("ffffffffffffffinvalid address: ", add)
	} else {
		fmt.Println("ffffffff")
	}
}
