package main

import (
	"log/slog"
	"os"

	"github.com/mikhaeris/sky-bank/notification_service/internal/app"
	"github.com/mikhaeris/sky-bank/notification_service/internal/config"
	"github.com/mikhaeris/sky-bank/notification_service/internal/handler"
	"github.com/mikhaeris/sky-bank/notification_service/internal/mailer"
	"github.com/mikhaeris/sky-bank/notification_service/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	mailer, err := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	notificationHandler := handler.NewNotificationHandler(logger, mailer)

	grpcServer := server.NewGRPCServer(notificationHandler, logger)

	err = app.RunGRPCServer(grpcServer, cfg.Server.Grpc.Addr, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
