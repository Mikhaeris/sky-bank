package server

import (
	"log/slog"

	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"

	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/grpc"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func NewGRPCServer(auth *handler.AuthHandler, challenge *handler.ChallengeHandler, logger *slog.Logger) *grpc.Server {
	logger.Info("new grpc server")

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptors(logger),
			interceptors.ErrorInterceptor(logger),
			interceptors.ValidationInterceptor(),
		),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
	grpcServer := grpc.NewServer(opts...)

	authv1.RegisterAuthServer(grpcServer, auth)
	authv1.RegisterOtpServiceServer(grpcServer, challenge)

	return grpcServer
}
