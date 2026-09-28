package postgresclient

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
)

const databasePingTimeout = 5 * time.Second

func OpenDB(storage *config.StorageConfig) (*pgxpool.Pool, error) {
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(storage.Username, storage.Password),
		Host:     net.JoinHostPort(storage.Host, storage.Port),
		Path:     "/" + storage.Database,
		RawQuery: "sslmode=disable",
	}

	db, err := pgxpool.New(context.Background(), dsn.String())
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()

	err = db.Ping(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
