package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/mikhaeris/sky-bank/gateway/internal/config"
	"github.com/mikhaeris/sky-bank/gateway/internal/registers"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	serverMux, clenup, err := registers.RegisterAll(cfg.Auth.Addr)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer clenup()

	err = http.ListenAndServe(cfg.Rest.Addr, serverMux)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
