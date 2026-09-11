package main

import (
	"log/slog"
	"os"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
	grpcclient "github.com/mikhaeris/sky-bank/auth_service/internal/clients/grpc"
	postgresclient "github.com/mikhaeris/sky-bank/auth_service/internal/clients/postgres"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/mikhaeris/sky-bank/auth_service/internal/utils"

	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	keys := utils.NewKeys(cfg.Jwt.PrivKeyPath, cfg.Jwt.AccessTokenTtl, logger)

	db, err := postgresclient.OpenDB(&cfg.Storage)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	notificationClient, closeNotificationClient, err := grpcclient.NewNotificationClient(cfg.Client.Grpc.Addr)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer closeNotificationClient()
	logger.Info("grpc notification_service connection established")

	identityRepositories := repository.NewAuthRepository(db)
	tokenRepositories := repository.NewTokenRepository(db)

	authService := service.NewAuthService(keys, logger, notificationClient, tokenRepositories, identityRepositories)

	authHandler := handler.NewAuthHandler(logger, authService)

	grpcServer := server.NewGRPCServer(authHandler, logger)

	err = app.RunGRPCServer(grpcServer, cfg.Server.Grpc.Addr, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
