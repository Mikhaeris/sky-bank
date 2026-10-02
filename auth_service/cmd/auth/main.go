package main

import (
	"log/slog"
	"os"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := app.Run(logger); err != nil {
		logger.Error("run auth service", "error", err)
		os.Exit(1)
	}
}
