package server

import (
	"log/slog"

	"github.com/mikhaeris/sky-bank/customer_service/internal/handler"
	customerv1 "github.com/mikhaeris/sky-bank/proto/gen/customer/v1"
	"google.golang.org/grpc"
)

func NewGRPCServer(customer *handler.CustomerHandler, logger *slog.Logger) *grpc.Server {
	logger.Info("new grpc server")
	grpcServer := grpc.NewServer()

	customerv1.RegisterCustomerServer(grpcServer, customer)

	return grpcServer
}
