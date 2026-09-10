package main

import (
	"log/slog"
	"os"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/postgresClient"
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

	db, err := postgresClient.OpenDB(&cfg.Storage)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	repositories := repository.NewAuthRepository(db)
	authService := service.NewAuthService(keys, logger, repositories)

	authHandler := handler.NewAuthHandler(logger, authService)

	grpcServer := server.NewGRPCServer(authHandler, logger)

	err = app.RunGRPCServer(grpcServer, cfg.Server.Grpc.Addr, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
