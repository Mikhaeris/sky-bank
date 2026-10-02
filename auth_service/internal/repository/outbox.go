package repository

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type OTPOutboxRepository struct {
	tracer trace.Tracer
	db     DBTX
}

func (or *OTPOutboxRepository) Claim(ctx context.Context, limit int, leaseDuration time.Duration) ([]domain.OTPOutbox, error) {
	if limit <= 0 || leaseDuration < time.Microsecond {
		return nil, fmt.Errorf("claim otp outbox: positive limit and lease of at least one microsecond required")
	}

	query := `
		WITH candidates AS (
			SELECT o.event_id
			FROM otp_outbox o
			WHERE o.expires_at > statement_timestamp()
			  AND (o.leased_until IS NULL OR o.leased_until <= statement_timestamp())
			ORDER BY o.expires_at, o.event_id
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE otp_outbox o
		SET leased_until = statement_timestamp() + $2 * INTERVAL '1 microsecond'
		FROM candidates
		WHERE o.event_id = candidates.event_id
		RETURNING o.event_id, o.challenge_id, o.encrypted_event,
		          COALESCE(o.trace_parent, ''), COALESCE(o.trace_state, ''),
		          o.expires_at, o.leased_until`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := or.db.Query(ctx, query, limit, leaseDuration.Microseconds())
	if err != nil {
		return nil, fmt.Errorf("claim otp outbox: %w", err)
	}
	defer rows.Close()

	var events []domain.OTPOutbox
	for rows.Next() {
		var event domain.OTPOutbox
		if err := rows.Scan(&event.EventID, &event.ChallengeID, &event.EncryptedEvent,
			&event.TraceParent, &event.TraceState, &event.ExpiresAt, &event.LeasedUntil); err != nil {
			return nil, fmt.Errorf("scan claimed otp outbox: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read claimed otp outbox: %w", err)
	}
	return events, nil
}

func (or *OTPOutboxRepository) DeleteClaimed(ctx context.Context, eventID uuid.UUID, leasedUntil time.Time) (bool, error) {
	ctx, span := or.tracer.Start(ctx, "otp_outbox.delete_claimed")
	defer span.End()

	query := `
		DELETE FROM otp_outbox
		WHERE event_id = $1 AND leased_until = $2
		  AND leased_until > statement_timestamp()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := or.db.Exec(ctx, query, eventID, leasedUntil)
	if err != nil {
		return false, fmt.Errorf("delete claimed otp outbox: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (or *OTPOutboxRepository) IsClaimed(ctx context.Context, eventID uuid.UUID, leasedUntil time.Time) (bool, error) {
	ctx, span := or.tracer.Start(ctx, "otp_outbox.is_claimed")
	defer span.End()

	query := `SELECT EXISTS (
		SELECT 1 FROM otp_outbox
		WHERE event_id = $1 AND leased_until = $2
		  AND leased_until > statement_timestamp()
		  AND expires_at > statement_timestamp()
		)`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var current bool
	err := or.db.QueryRow(ctx, query, eventID, leasedUntil).Scan(&current)
	if err != nil {
		return false, fmt.Errorf("check claimed otp outbox: %w", err)
	}
	return current, nil
}

func (or *OTPOutboxRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM otp_outbox
		WHERE expires_at <= statement_timestamp()`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tag, err := or.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired otp outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func NewOTPOutboxRepository(
	tracer trace.Tracer,
	db DBTX,
) *OTPOutboxRepository {
	return &OTPOutboxRepository{
		tracer: tracer,
		db:     db,
	}
}

func (or *OTPOutboxRepository) Insert(ctx context.Context, dto domain.OTPOutbox) error {
	ctx, span := or.tracer.Start(ctx, "otp_outbox.insert")
	defer span.End()

	query := `
		INSERT INTO otp_outbox (
			event_id,
			challenge_id,
			encrypted_event,
			trace_parent,
			trace_state,
			expires_at
		 )
		VALUES ($1, $2, $3, $4, $5, $6)`

	args := []any{
		dto.EventID,
		dto.ChallengeID,
		dto.EncryptedEvent,
		dto.TraceParent,
		dto.TraceState,
		dto.ExpiresAt,
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := or.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert otp outbox: %w", err)
	}

	return nil
}

func (or *OTPOutboxRepository) DeleteByChallengeID(ctx context.Context, oldChallengeID uuid.UUID) error {
	ctx, span := or.tracer.Start(ctx, "otp_outbox.delete")
	defer span.End()

	query := `
		DELETE FROM otp_outbox
		WHERE challenge_id = $1`

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := or.db.Exec(ctx, query, oldChallengeID)
	if err != nil {
		return fmt.Errorf("delete otp outbox by challenge_id: %w", err)
	}

	return nil
}
