package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type TokenRepository struct {
	DB *sql.DB
}

func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{
		DB: db,
	}
}

func (t *TokenRepository) Insert(ctx context.Context, token *domain.Token) error {
	query := `
		INSERT INTO tokens (hash, identities_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.IdentityId, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

func (t *TokenRepository) DeleteAllForUser(ctx context.Context, scope string, userId uuid.UUID) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND identities_id = $2`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, scope, userId)
	return err
}

func (t *TokenRepository) DeleteTokenForUser(ctx context.Context, tokenPlaintext string, userId uuid.UUID) error {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		DELETE FROM tokens
		WHERE hash = $1 AND identities_id = $2`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, tokenHash[:], userId)
	return err

}
