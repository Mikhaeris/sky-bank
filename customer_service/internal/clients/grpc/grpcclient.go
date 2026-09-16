package grpcclient

import (
	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	notificationClientv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewNotificationClient(addr string) (notificationClientv1.NotificationServiceClient, func() error, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := notificationClientv1.NewNotificationServiceClient(conn)

	return client, conn.Close, nil
}

func NewAuthClient(addr string) (authv1.AuthClient, func() error, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := authv1.NewAuthClient(conn)

	return client, conn.Close, nil
}
