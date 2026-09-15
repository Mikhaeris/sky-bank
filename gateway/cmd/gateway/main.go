package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/mikhaeris/sky-bank/gateway/internal/config"
	jwt "github.com/mikhaeris/sky-bank/gateway/internal/lib"
	"github.com/mikhaeris/sky-bank/gateway/internal/registers"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	tokenVerifier, err := jwt.NewTokenVerifier(cfg.Jwt.PubKeyPath)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	serverMux, cleanup, err := registers.RegisterAll(cfg.Auth.Addr, tokenVerifier)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer cleanup()

	mux := http.NewServeMux()

	mux.Handle(
		"/api/v1/",
		http.StripPrefix("/api/v1", serverMux),
	)

	err = http.ListenAndServe(cfg.Rest.Addr, mux)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
