package grpcclient

import (
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
