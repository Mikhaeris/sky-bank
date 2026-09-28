package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type TxStore interface {
	ChallengeRepository() *ChallengeRepository
	IdentityRepository() *IdentityRepository
	OTPLimitRepository() *OTPLimitRepository
	SessionRepository() *SessionRepository
}

type DataStore interface {
	TxStore
	Atomic(
		context.Context,
		func(context.Context, TxStore) error,
	) error
}

type repositories struct {
	tracer trace.Tracer
	db     DBTX
}

func (r *repositories) ChallengeRepository() *ChallengeRepository {
	return NewChallengeRepository(r.tracer, r.db)
}

func (r *repositories) IdentityRepository() *IdentityRepository {
	return NewIdentityRepository(r.db)
}

func (r *repositories) OTPLimitRepository() *OTPLimitRepository {
	return NewOTPLimitRepository(r.tracer, r.db)
}

func (r *repositories) SessionRepository() *SessionRepository {
	return NewSessionRepository(r.db)
}

type dataStore struct {
	*repositories
	pool *pgxpool.Pool
}

func NewDataStore(tracer trace.Tracer, pool *pgxpool.Pool) DataStore {
	return &dataStore{
		repositories: &repositories{
			tracer: tracer,
			db:     pool,
		},
		pool: pool,
	}
}

func (s *dataStore) Atomic(
	ctx context.Context,
	fn func(context.Context, TxStore) error,
) (err error) {
	ctx, span := s.tracer.Start(ctx, "db.transaction")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}()

	dbTx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer dbTx.Rollback(ctx)

	txStore := &repositories{
		tracer: s.tracer,
		db:     dbTx,
	}

	if err = fn(ctx, txStore); err != nil {
		return err
	}

	if err = dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
