package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type OTPLimitRepository struct {
	tracer trace.Tracer
	db     DBTX
}

func NewOTPLimitRepository(
	tracer trace.Tracer,
	db DBTX,
) *OTPLimitRepository {
	return &OTPLimitRepository{
		tracer: tracer,
		db:     db,
	}
}

func (r *OTPLimitRepository) Lock(
	ctx context.Context,
	destination string,
	channel domain.OtpChannel,
	purpose domain.CodePurpose,
) (domain.OTPLimitState, error) {
	ctx, span := r.tracer.Start(ctx, "otp_limit.lock")
	defer span.End()

	query := `
		INSERT INTO otp_limits (destination, channel, purpose)
		VALUES ($1, $2, $3)
		ON CONFLICT (destination, channel, purpose)
		DO UPDATE SET updated_at = otp_limits.updated_at
		RETURNING destination, channel, purpose,
		          last_issued_at, issue_window_started_at, issue_count,
		          failure_window_started_at, failure_count, blocked_until, updated_at`

	args := []any{destination, channel, purpose}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var limit domain.OTPLimitState
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&limit.Destination,
		&limit.Channel,
		&limit.Purpose,
		&limit.LastIssuedAt,
		&limit.IssueWindowStartedAt,
		&limit.IssueCount,
		&limit.FailureWindowStartedAt,
		&limit.FailureCount,
		&limit.BlockedUntil,
		&limit.UpdatedAt,
	)
	if err != nil {
		return domain.OTPLimitState{}, fmt.Errorf("lock otp limit: %w", err)
	}

	return limit, nil
}

func (r *OTPLimitRepository) DeleteInactive(
	ctx context.Context,
	cutoff time.Time,
	now time.Time,
) (int64, error) {
	query := `
		DELETE FROM otp_limits
		WHERE updated_at < $1
		  AND (blocked_until IS NULL OR blocked_until <= $2)`

	args := []any{cutoff, now}

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete inactive otp limits: %w", err)
	}

	return result.RowsAffected(), nil
}

func (r *OTPLimitRepository) RecordFailure(ctx context.Context, state domain.OTPLimitState) error {
	query := `
		UPDATE otp_limits
		SET failure_window_started_at = $4,
		    failure_count = $5,
		    blocked_until = $6,
		    updated_at = $7
		WHERE destination = $1 AND channel = $2 AND purpose = $3`

	args := []any{
		state.Destination,
		state.Channel,
		state.Purpose,
		state.FailureWindowStartedAt,
		state.FailureCount,
		state.BlockedUntil,
		state.UpdatedAt,
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("record otp failure: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("record otp failure: %w", ErrRecordNotFound)
	}
	return nil
}

func (r *OTPLimitRepository) RecordIssue(ctx context.Context, state domain.OTPLimitState) error {
	ctx, span := r.tracer.Start(ctx, "otp_limit.record_issue")
	defer span.End()

	query := `
		UPDATE otp_limits
		SET last_issued_at = $4,
		    issue_window_started_at = $5,
		    issue_count = $6,
		    updated_at = $7
		WHERE destination = $1 AND channel = $2 AND purpose = $3`

	args := []any{
		state.Destination,
		state.Channel,
		state.Purpose,
		state.LastIssuedAt,
		state.IssueWindowStartedAt,
		state.IssueCount,
		state.UpdatedAt,
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("record otp issue: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("record otp issue: %w", ErrRecordNotFound)
	}
	return nil
}
