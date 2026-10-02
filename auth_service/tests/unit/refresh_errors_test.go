package unit_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type refreshRow struct{ err error }

func (r refreshRow) Scan(...any) error { return r.err }

type refreshDB struct {
	queryErr error
	execErr  error
}

func (db refreshDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return refreshRow{err: db.queryErr}
}
func (refreshDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}
func (db refreshDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, db.execErr
}

type refreshStore struct{ db refreshDB }

var testTracer = noop.NewTracerProvider().Tracer("test")

func (s refreshStore) SessionRepository() *repository.SessionRepository {
	return repository.NewSessionRepository(testTracer, s.db)
}
func (refreshStore) IdentityRepository() *repository.IdentityRepository {
	panic("unexpected identity lookup")
}
func (refreshStore) ChallengeRepository() *repository.ChallengeRepository {
	panic("unexpected challenge lookup")
}
func (refreshStore) OTPLimitRepository() *repository.OTPLimitRepository {
	panic("unexpected OTP limit lookup")
}
func (refreshStore) OTPOutboxRepository() *repository.OTPOutboxRepository {
	panic("unexpected outbox lookup")
}
func (refreshStore) Atomic(context.Context, func(context.Context, repository.TxStore) error) error {
	panic("unexpected transaction")
}

func TestRefreshTokensDistinguishesMissingSessionFromDatabaseFailure(t *testing.T) {
	dbErr := errors.New("database unavailable")
	tests := []struct {
		name        string
		queryErr    error
		wantInvalid bool
		wantCode    codes.Code
	}{
		{name: "unknown token", queryErr: pgx.ErrNoRows, wantInvalid: true, wantCode: codes.Unauthenticated},
		{name: "database failure", queryErr: dbErr, wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := refreshStore{db: refreshDB{queryErr: tt.queryErr}}
			svc := service.NewIdentityService(testTracer, nil, nil, nil, store)
			_, err := svc.RefreshTokens(context.Background(), domain.TokensDTO{Refersh: "test-token"})
			if err == nil {
				t.Fatal("expected an error")
			}
			if got := errors.Is(err, service.ErrInvalidRefreshToken); got != tt.wantInvalid {
				t.Fatalf("invalid-token classification = %t, want %t: %v", got, tt.wantInvalid, err)
			}
			if !tt.wantInvalid && !errors.Is(err, dbErr) {
				t.Fatalf("database cause was lost: %v", err)
			}

			var logOutput bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logOutput, nil))
			h := handler.NewAuthHandler(testTracer, svc, nil)
			_, publicErr := interceptors.ErrorInterceptor(logger)(
				context.Background(), nil,
				&grpc.UnaryServerInfo{FullMethod: "/auth.v1.Auth/RefreshTokens"},
				func(ctx context.Context, _ any) (any, error) {
					return h.RefreshTokens(ctx, &authv1.RefreshTokensRequest{RefreshToken: "test-token"})
				},
			)
			if status.Code(publicErr) != tt.wantCode {
				t.Fatalf("public code = %v, want %v", status.Code(publicErr), tt.wantCode)
			}
			if tt.wantInvalid {
				if logOutput.Len() != 0 {
					t.Fatalf("expected rejection was logged as a server error: %q", logOutput.String())
				}
			} else {
				if status.Convert(publicErr).Message() != "internal error" {
					t.Fatalf("internal cause leaked to client: %v", publicErr)
				}
				if !strings.Contains(logOutput.String(), dbErr.Error()) {
					t.Fatalf("internal cause missing from log: %q", logOutput.String())
				}
			}
		})
	}
}

func TestRefreshRotationClassifiesConflictAndPreservesDatabaseFailure(t *testing.T) {
	repo := repository.NewSessionRepository(testTracer, refreshDB{})
	if err := repo.Update(context.Background(), domain.Session{}, nil); !errors.Is(err, repository.ErrEditConflict) {
		t.Fatalf("zero updated rows should be an edit conflict: %v", err)
	}

	cause := errors.New("database unavailable")
	repo = repository.NewSessionRepository(testTracer, refreshDB{execErr: cause})
	if err := repo.Update(context.Background(), domain.Session{}, nil); !errors.Is(err, cause) {
		t.Fatalf("database cause was lost: %v", err)
	}
}
