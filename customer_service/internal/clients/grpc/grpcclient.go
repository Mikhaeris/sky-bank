package grpcclient

import (
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewOtpServiceClient(addr string) (authv1.OtpServiceClient, func() error, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := authv1.NewOtpServiceClient(conn)

	return client, conn.Close, nil
}
