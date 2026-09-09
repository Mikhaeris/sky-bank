package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mikhaeris/bank-test/auth_service/internal/handler"
	"github.com/mikhaeris/bank-test/auth_service/internal/repository"
	"github.com/mikhaeris/bank-test/auth_service/internal/service"

	_ "github.com/lib/pq"
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
		port: 4000,
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

	logger.Info("starting server", "addr", cfg.port)
	err = http.ListenAndServe(":4000", routes(handler))
	if err != nil {
		fmt.Print(err)
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
