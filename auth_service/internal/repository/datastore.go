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
	OTPOutboxRepository() *OTPOutboxRepository
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
	return NewIdentityRepository(r.tracer, r.db)
}

func (r *repositories) OTPLimitRepository() *OTPLimitRepository {
	return NewOTPLimitRepository(r.tracer, r.db)
}

func (r *repositories) SessionRepository() *SessionRepository {
	return NewSessionRepository(r.tracer, r.db)
}

func (r *repositories) OTPOutboxRepository() *OTPOutboxRepository {
	return NewOTPOutboxRepository(r.tracer, r.db)
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

	beginCtx, beginSpan := s.tracer.Start(ctx, "db.transaction.begin")
	dbTx, err := s.pool.Begin(beginCtx)
	beginSpan.End()

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

	commitCtx, commitSpan := s.tracer.Start(ctx, "db.transaction.commit")

	err = dbTx.Commit(commitCtx)

	commitSpan.End()

	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
