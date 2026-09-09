package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"

	_ "github.com/lib/pq"
	pb "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"google.golang.org/grpc"
)

type config struct {
	port int
	db   struct {
		dsn string
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config{
		port: 9001,
	}

	cfg.db.dsn = "postgres://mikhaeris:qwerty@:5432/bank?sslmode=disable"

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	repo := repository.NewAuthRepository(db)
	service := service.NewAuthService(logger, repo)
	handler := handler.NewAuthHandler(logger, service)

	lis, err := net.Listen("tcp", "localhost:9001")
	if err != nil {
		log.Println("error starting tcp listener: ", err)
		os.Exit(1)
	}
	logger.Info("starting server", "addr", cfg.port)
	grpcServer := grpc.NewServer()

	pb.RegisterAuthSericeServer(grpcServer, handler)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Error("error serving grpc", err)
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
