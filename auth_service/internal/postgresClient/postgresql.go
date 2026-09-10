package postgresClient

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
)

func OpenDB(storage *config.StorageConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		storage.Username,
		storage.Password,
		storage.Host,
		storage.Port,
		storage.Database,
	)

	db, err := sql.Open("postgres", dsn)
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
