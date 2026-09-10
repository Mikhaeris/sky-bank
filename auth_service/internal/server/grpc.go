package server

import (
	"log/slog"

	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"

	pb "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"google.golang.org/grpc"
)

func NewGRPCServer(auth *handler.AuthHandler, logger *slog.Logger) *grpc.Server {
	logger.Info("new grpc server")
	grpcServer := grpc.NewServer()

	pb.RegisterAuthSericeServer(grpcServer, auth)

	return grpcServer
}
