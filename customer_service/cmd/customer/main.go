package main

import (
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	"github.com/mikhaeris/sky-bank/customer_service/internal/app"
	grpcclient "github.com/mikhaeris/sky-bank/customer_service/internal/clients/grpc"
	postgresclient "github.com/mikhaeris/sky-bank/customer_service/internal/clients/postgers"
	"github.com/mikhaeris/sky-bank/customer_service/internal/config"
	"github.com/mikhaeris/sky-bank/customer_service/internal/handler"
	"github.com/mikhaeris/sky-bank/customer_service/internal/repository"
	"github.com/mikhaeris/sky-bank/customer_service/internal/server"
	"github.com/mikhaeris/sky-bank/customer_service/internal/service"
	"github.com/mikhaeris/sky-bank/pkg/kafka"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	db, err := postgresclient.OpenDB(&cfg.Storage)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	producer, err := kafka.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		logger.Error("create kafka producer", "error", err)
		os.Exit(1)
	}

	otpServiceClient, closeOtpServiceClient, err := grpcclient.NewOtpServiceClient(cfg.Client.Grpc.Auth.Addr)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer closeOtpServiceClient()
	logger.Info("grpc auth_service connection established")

	customerRepository := repository.NewCustomerRepository(db)
	customerService := service.NewCustomerService(customerRepository, producer, otpServiceClient)
	customerHandler := handler.NewCustomerHandler(customerService)

	grpcServer := server.NewGRPCServer(customerHandler, logger)

	err = app.RunGRPCServer(grpcServer, cfg.Server.Grpc.Addr, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
