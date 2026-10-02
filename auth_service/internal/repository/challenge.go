package repository

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type ChallengeRepository struct {
	tracer trace.Tracer
	db     DBTX
}

func NewChallengeRepository(
	tracer trace.Tracer,
	db DBTX,
) *ChallengeRepository {
	return &ChallengeRepository{
		tracer: tracer,
		db:     db,
	}
}

func (cr *ChallengeRepository) Upster(ctx context.Context, otp *domain.Challenge) (uuid.UUID, error) {
	ctx, span := cr.tracer.Start(ctx, "challenge.upsert")
	defer span.End()

	query := `
	INSERT INTO challenges (
    	id,
	    destination,
	    channel,
	    purpose,
	    code_hash,
	    expires_at
	)
	VALUES ($1, $2, $3, $4, $5, $6)

	ON CONFLICT (destination, channel, purpose)
	DO UPDATE SET
	    id = EXCLUDED.id,
	    code_hash = EXCLUDED.code_hash,
	    expires_at = EXCLUDED.expires_at,
	    failed_attempts = 0
	RETURNING OLD.id`

	args := []any{
		otp.ID,
		otp.Destination,
		otp.Channel,
		otp.Purpose,
		otp.CodeHash,
		otp.ExpiresAt,
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var oldChallengeID pgtype.UUID
	err := cr.db.QueryRow(ctx, query, args...).Scan(
		&oldChallengeID,
	)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("upsert challenge: %w", err)
	}

	if !oldChallengeID.Valid {
		return uuid.Nil(), nil
	}

	return uuid.UUID(oldChallengeID.Bytes), nil
}

func (cr *ChallengeRepository) GetByChallengeId(ctx context.Context, challengeID uuid.UUID) (domain.Challenge, error) {
	return cr.getByChallengeID(ctx, challengeID, false)
}

func (cr *ChallengeRepository) GetByChallengeIdForUpdate(ctx context.Context, challengeID uuid.UUID) (domain.Challenge, error) {
	return cr.getByChallengeID(ctx, challengeID, true)
}

func (cr *ChallengeRepository) getByChallengeID(ctx context.Context, challengeID uuid.UUID, forUpdate bool) (domain.Challenge, error) {
	ctx, span := cr.tracer.Start(ctx, "challenge.get")
	defer span.End()

	query := `
		SELECT id, destination, channel, purpose, code_hash, expires_at, failed_attempts
		FROM challenges
		WHERE id = $1 AND expires_at > NOW()`
	if forUpdate {
		query += " FOR UPDATE"
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	code := domain.Challenge{}

	err := cr.db.QueryRow(ctx, query, challengeID).Scan(
		&code.ID,
		&code.Destination,
		&code.Channel,
		&code.Purpose,
		&code.CodeHash,
		&code.ExpiresAt,
		&code.FailedAttempts,
	)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return domain.Challenge{}, fmt.Errorf("find challenge: %w", ErrRecordNotFound)
		default:
			return domain.Challenge{}, fmt.Errorf("find challenge: %w", err)
		}
	}

	return code, nil
}

func (cr *ChallengeRepository) IncrementFailedAttempts(ctx context.Context, challengeID uuid.UUID) error {
	ctx, span := cr.tracer.Start(ctx, "challenge.increment")
	defer span.End()

	query := `
		UPDATE challenges
		SET failed_attempts = failed_attempts + 1
		WHERE id = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := cr.db.Exec(ctx, query, challengeID)
	if err != nil {
		return fmt.Errorf("increment challenge failures: %w", err)
	}

	if result.RowsAffected() != 1 {
		return fmt.Errorf("increment challenge failures: %w", ErrRecordNotFound)
	}

	return nil
}

func (cr *ChallengeRepository) DeleteByChallengeId(ctx context.Context, challengeID uuid.UUID) error {
	ctx, span := cr.tracer.Start(ctx, "challenge.delete")
	defer span.End()

	query := `
		DELETE FROM challenges
		WHERE id = $1 AND expires_at > NOW()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := cr.db.Exec(ctx, query, challengeID)
	if err != nil {
		return fmt.Errorf("delete challenge: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("delete challenge: %w", ErrRecordNotFound)
	}

	return nil
}

func (cr *ChallengeRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	query := `DELETE FROM challenges
		WHERE expires_at <= $1`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := cr.db.Exec(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired challenges: %w", err)
	}

	return result.RowsAffected(), nil
}
