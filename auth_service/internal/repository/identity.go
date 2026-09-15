package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type IdentityRepository struct {
	DB *sql.DB
}

func NewIdentityRepository(db *sql.DB) *IdentityRepository {
	return &IdentityRepository{
		DB: db,
	}
}

func (ir *IdentityRepository) GetByID(ctx context.Context, identityID uuid.UUID) (domain.Identity, error) {
	query := `
		SELECT id, email, version
		FROM identities
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var identity domain.Identity

	err := ir.DB.QueryRowContext(ctx, query, identityID).Scan(
		&identity.ID,
		&identity.Email,
		&identity.Version,
	)
	if err != nil {
		return domain.Identity{}, err
	}

	return identity, nil
}

func (ir *IdentityRepository) GetOrCreateByEmail(ctx context.Context, email string) (domain.Identity, error) {
	tx, err := ir.DB.Begin()
	if err != nil {
		return domain.Identity{}, err
	}
	defer tx.Rollback()

	id := uuid.New()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO identities (id, email, version)
		VALUES ($1, $2, 1)
		ON CONFLICT (email) DO NOTHING`

	args := []any{id, email}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.Identity{}, err
	}

	query = `
		SELECT id, email, version
		FROM identities
		WHERE email = $1`

	var identity domain.Identity

	err = tx.QueryRowContext(ctx, query, email).Scan(
		&identity.ID,
		&identity.Email,
		&identity.Version,
	)
	if err != nil {
		return domain.Identity{}, err
	}

	err = tx.Commit()
	if err != nil {
		return domain.Identity{}, err
	}

	return identity, err
}
