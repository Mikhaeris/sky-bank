package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type SessionRepository struct {
	tracer trace.Tracer
	db     DBTX
}

func NewSessionRepository(
	tracer trace.Tracer,
	db DBTX,
) *SessionRepository {
	return &SessionRepository{
		tracer: tracer,
		db:     db,
	}
}

func (sr *SessionRepository) Insert(ctx context.Context, session domain.Session) error {
	ctx, span := sr.tracer.Start(ctx, "session.insert")
	defer span.End()

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

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := sr.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (sr *SessionRepository) GetById(ctx context.Context, sessionId uuid.UUID) (domain.Session, error) {
	ctx, span := sr.tracer.Start(ctx, "session.get_by_id")
	defer span.End()

	query := `
		SELECT id, identity_id, refresh_token_hash, created_at, expires_at, last_used_at
		FROM sessions
		WHERE id = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var session domain.Session

	err := sr.db.QueryRow(ctx, query, sessionId).Scan(
		&session.ID,
		&session.IdentityID,
		&session.RefreshTokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("find id session: %w", ErrRecordNotFound)
		}
		return domain.Session{}, fmt.Errorf("find id session: %w", err)
	}

	return session, nil
}

func (sr *SessionRepository) GetByRefreshTokenHash(ctx context.Context, refreshTokenHash []byte) (domain.Session, error) {
	ctx, span := sr.tracer.Start(ctx, "session.get_by_token_hash")
	defer span.End()

	query := `
		SELECT id, identity_id, refresh_token_hash, created_at, expires_at, last_used_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var session domain.Session

	err := sr.db.QueryRow(ctx, query, refreshTokenHash).Scan(
		&session.ID,
		&session.IdentityID,
		&session.RefreshTokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("find refresh session: %w", ErrRecordNotFound)
		}
		return domain.Session{}, fmt.Errorf("find refresh session: %w", err)
	}

	return session, nil
}

func (sr *SessionRepository) GetSessionsByIdentityID(ctx context.Context, identityID uuid.UUID) ([]*v1.Session, error) {
	ctx, span := sr.tracer.Start(ctx, "session.get_by_identity_id")
	defer span.End()

	query := `
		SELECT id, created_at, expires_at, last_used_at
		FROM sessions
		WHERE identity_id = $1`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := sr.db.Query(ctx, query, identityID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []*v1.Session{}

	for rows.Next() {
		var session v1.Session
		var createdAt, expiresAt, lastUsedAt time.Time

		err := rows.Scan(
			&session.Id,
			&createdAt,
			&expiresAt,
			&lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		session.CreatedAt = createdAt.Format(time.RFC3339Nano)
		session.ExpiresAt = expiresAt.Format(time.RFC3339Nano)
		session.LastUsedAt = lastUsedAt.Format(time.RFC3339Nano)

		sessions = append(sessions, &session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func (sr *SessionRepository) Update(ctx context.Context, session domain.Session, oldTokenHash []byte) error {
	ctx, span := sr.tracer.Start(ctx, "session.update")
	defer span.End()

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

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := sr.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("rotate refresh session: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("rotate refresh session: %w", ErrEditConflict)
	}

	return nil
}

func (sr *SessionRepository) DeleteSessionByID(ctx context.Context, sessionID, identityID uuid.UUID) error {
	ctx, span := sr.tracer.Start(ctx, "session.delete")
	defer span.End()

	query := `
		DELETE FROM sessions
		WHERE id = $1 AND identity_id = $2`

	agrs := []any{sessionID, identityID}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := sr.db.Exec(ctx, query, agrs...)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("delete session: %w", ErrRecordNotFound)
	}

	return nil
}

func (sr *SessionRepository) DeleteOtherSessionsByID(ctx context.Context, sessionID, identityID uuid.UUID) error {
	ctx, span := sr.tracer.Start(ctx, "session.delete_other")
	defer span.End()

	query := `
		DELETE FROM sessions
		WHERE identity_id = $1
		  AND id <> $2`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := sr.db.Exec(ctx, query, identityID, sessionID)
	if err != nil {
		return fmt.Errorf("delete other sessions: %w", err)
	}

	return nil
}
