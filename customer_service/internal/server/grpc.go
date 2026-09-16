package server

import (
	"log/slog"

	customerv1 "github.com/mikhaeris/sky-bank/customer_service/api/customer/v1"
	"github.com/mikhaeris/sky-bank/customer_service/internal/handler"
	"google.golang.org/grpc"
)

func NewGRPCServer(customer *handler.CustomerHandler, logger *slog.Logger) *grpc.Server {
	logger.Info("new grpc server")
	grpcServer := grpc.NewServer()

	customerv1.RegisterCustomerServer(grpcServer, customer)
	customerv1.RegisterCustomerPrivateServer(grpcServer, customer)

	return grpcServer
}
