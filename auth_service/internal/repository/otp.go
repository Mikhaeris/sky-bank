package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type OtpRepository struct {
	DB *sql.DB
}

func NewTokenRepository(db *sql.DB) *OtpRepository {
	return &OtpRepository{
		DB: db,
	}
}

func (t *OtpRepository) Insert(ctx context.Context, otp *domain.OneTimeCode) error {
	query := `
		INSERT INTO one_time_codes (email, purpose, code_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email, purpose)
		DO UPDATE SET
		    code_hash = EXCLUDED.code_hash,
		    expires_at = EXCLUDED.expires_at;`

	args := []any{otp.Email, otp.Purpose, otp.CodeHash, otp.ExpiresAt}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

func (t *OtpRepository) GetByEmail(ctx context.Context, email string, purpose domain.CodePurpose) (domain.OneTimeCode, error) {
	query := `
		SELECT email, purpose, code_hash, expires_at
		FROM one_time_codes
		WHERE email = $1 AND purpose = $2`

	args := []any{email, purpose}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	code := domain.OneTimeCode{}

	err := t.DB.QueryRowContext(ctx, query, args...).Scan(
		&code.Email,
		&code.Purpose,
		&code.CodeHash,
		&code.ExpiresAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return domain.OneTimeCode{}, ErrRecordNotFound
		default:
			return domain.OneTimeCode{}, err
		}
	}

	return code, nil
}

func (t *OtpRepository) DeleteByEmail(ctx context.Context, email string, purpose domain.CodePurpose) error {
	query := `
		DELETE FROM one_time_codes
		WHERE email = $1 AND purpose = $2`

	args := []any{email, purpose}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

// func (t *TokenRepository) DeleteAllForUser(ctx context.Context, scope string, userId uuid.UUID) error {
// 	query := `
// 		DELETE FROM tokens
// 		WHERE scope = $1 AND identities_id = $2`

// 	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()

// 	_, err := t.DB.ExecContext(ctx, query, scope, userId)
// 	return err
// }

// func (t *TokenRepository) DeleteTokenForUser(ctx context.Context, tokenPlaintext string, userId uuid.UUID) error {
// 	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

// 	query := `
// 		DELETE FROM tokens
// 		WHERE hash = $1 AND identities_id = $2`

// 	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()

// 	_, err := t.DB.ExecContext(ctx, query, tokenHash[:], userId)
// 	return err

// }
