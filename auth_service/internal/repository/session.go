package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"uuid"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type SessionRepository struct {
	DB *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		DB: db,
	}
}

func (sr *SessionRepository) Insert(ctx context.Context, session domain.Session) error {
	query := `
		INSERT INTO sessions (id, identity_id, refresh_token_hash, created_at, expires_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	args := []any{
		session.ID,
		session.IdentityID,
		session.RefreshTokenHash,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastUsedAt,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := sr.DB.ExecContext(ctx, query, args...)
	return err
}

func (sr *SessionRepository) GetByRefreshTokenHash(ctx context.Context, refreshTokenHash []byte) (domain.Session, error) {
	query := `
		SELECT id, identity_id, refresh_token_hash, created_at, expires_at, last_used_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var session domain.Session

	err := sr.DB.QueryRowContext(ctx, query, refreshTokenHash).Scan(
		&session.ID,
		&session.IdentityID,
		&session.RefreshTokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastUsedAt,
	)
	if err != nil {
		return domain.Session{}, err
	}

	return session, nil
}

func (sr *SessionRepository) GetSessionsByIdentityID(ctx context.Context, identityID uuid.UUID) ([]*v1.Session, error) {
	query := `
		SELECT id, created_at, expires_at, last_used_at
		FROM sessions
		WHERE identity_id = $1`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := sr.DB.QueryContext(ctx, query, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []*v1.Session{}

	for rows.Next() {
		var session v1.Session

		err := rows.Scan(
			&session.Id,
			&session.CreatedAt,
			&session.ExpiresAt,
			&session.LastUsedAt,
		)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, &session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (sr *SessionRepository) Update(ctx context.Context, session domain.Session, oldTokenHash []byte) error {
	query := `
		UPDATE sessions
		SET refresh_token_hash = $1, created_at = $2, expires_at = $3, last_used_at = $4
		WHERE id = $5 AND refresh_token_hash = $6`

	args := []any{
		session.RefreshTokenHash,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastUsedAt,
		session.ID,
		oldTokenHash,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := sr.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("invalid session")
	}

	return nil
}

func (sr *SessionRepository) DeleteSessionByID(ctx context.Context, sessionID, identityID uuid.UUID) error {
	query := `
		DELETE FROM sessions
		WHERE id = $1 AND identity_id = $2`

	agrs := []any{sessionID, identityID}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := sr.DB.ExecContext(ctx, query, agrs...)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

func (sr *SessionRepository) DeleteOtherSessionsByID(ctx context.Context, sessionID, identityID uuid.UUID) error {
	query := `
		DELETE FROM sessions
		WHERE identity_id = $1
		  AND id <> $2`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := sr.DB.ExecContext(ctx, query, identityID, sessionID)
	if err != nil {
		return err
	}

	return nil
}
