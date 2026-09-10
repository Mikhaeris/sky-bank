package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	pb "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var GrpcClient pb.AuthSericeClient

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	host := "auth"
	port := "9000"

	addr := fmt.Sprintf("%s:%s", host, port)

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("could not create grpc client",
			"error", err,
		)
		os.Exit(1)
	}
	defer conn.Close()

	GrpcClient = pb.NewAuthSericeClient(conn)

	logger.Info("grpc client created",
		"addr", addr,
	)

	conn.Connect()

	handler := AuthHandler{}

	logger.Info("starting server", "addr", 8081)
	err = http.ListenAndServe(":8081", routes(&handler))
	if err != nil {
		logger.Error("error start http server")
		os.Exit(1)
	}
}
