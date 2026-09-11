package server

import (
	"log/slog"

	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"github.com/mikhaeris/sky-bank/notification_service/internal/handler"
	"google.golang.org/grpc"
)

func NewGRPCServer(noti *handler.NotificationHandler, logger *slog.Logger) *grpc.Server {
	logger.Info("new grpc server")
	grpcServer := grpc.NewServer()

	notificationv1.RegisterNotificationServiceServer(grpcServer, noti)

	return grpcServer
}
