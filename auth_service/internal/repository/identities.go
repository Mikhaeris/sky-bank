package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
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

func (ir *IdentityRepository) Insert(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO identities (id, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	args := []any{user.Id, user.Email, user.PasswordHash, user.Activated}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var pqErr *pq.Error

	err := ir.DB.QueryRowContext(ctx, query, args...).Scan(&user.Id)
	if err != nil {
		switch {
		case errors.As(err, &pqErr) &&
			pqErr.Code == "23505" &&
			pqErr.Constraint == "identities_email_key":
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (ir *IdentityRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE identities
		SET email = $1, password_hash = $2, activated = $3, version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING version`

	args := []any{
		user.Email,
		user.PasswordHash,
		user.Activated,
		user.Id,
		user.Version,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var pqErr *pq.Error

	err := ir.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.As(err, &pqErr) &&
			pqErr.Code == "23505" &&
			pqErr.Constraint == "identities_email_key":
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (ir *IdentityRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, activated, version
		FROM identities
		WHERE email = $1`

	var user domain.User

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := ir.DB.QueryRowContext(ctx, query, email).Scan(
		&user.Id,
		&user.Email,
		&user.PasswordHash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (ir *IdentityRepository) GetForToken(ctx context.Context, tokenScope, tokenPlaintext string) (*domain.User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT identities.id, identities.email, identities.password_hash, identities.activated, identities.version
		FROM identities
		INNER JOIN tokens
		ON identities.id = tokens.identities_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user domain.User

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err := ir.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.Id,
		&user.Email,
		&user.PasswordHash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
