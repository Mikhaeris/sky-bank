package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/hasher"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.opentelemetry.io/otel/trace/noop"
)

const testCode = "123456"

type otpTestEnv struct {
	ctx     context.Context
	pool    *pgxpool.Pool
	store   repository.DataStore
	hasher  *hasher.CodeHasher
	cipher  *hasher.EventCipher
	service *service.ChallengeService
	limits  config.OtpLimits
}

func newOTPTestEnv(t *testing.T) *otpTestEnv {
	t.Helper()

	ctx := context.Background()
	migrations := filepath.Join("..", "..", "migrations")
	container, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase("auth_test"),
		postgres.WithUsername("auth_test"),
		postgres.WithPassword("test_password"),
		postgres.WithOrderedInitScripts(
			filepath.Join(migrations, "000001_create_identities_table.up.sql"),
			filepath.Join(migrations, "000002_create_challenges_table.up.sql"),
			filepath.Join(migrations, "000003_create_otp_limits.up.sql"),
			filepath.Join(migrations, "000004_create_sessions_table.sql.up.sql"),
			filepath.Join(migrations, "000005_create_outbox_table.up.sql"),
		),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	secretPath := filepath.Join(t.TempDir(), "otp-secret")
	if err := os.WriteFile(secretPath, []byte("01234567890123456789012345678901"), 0600); err != nil {
		t.Fatal(err)
	}
	codeHasher, err := hasher.NewCodeHasher(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	outboxKeyPath := filepath.Join(t.TempDir(), "outbox-secret")
	if err := os.WriteFile(outboxKeyPath, []byte("abcdefghijklmnopqrstuvwxyz012345"), 0600); err != nil {
		t.Fatal(err)
	}
	eventCipher, err := hasher.NewEventCipher(outboxKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	limits := config.OtpLimits{
		InitialIssues: 5, InitialCooldown: 15 * time.Second, MaxCooldown: 5 * time.Minute,
		IssueResetAfter:         30 * time.Minute,
		MaxAttemptsPerChallenge: 5,
	}
	tracer := noop.NewTracerProvider().Tracer("test")
	store := repository.NewDataStore(tracer, pool)
	return &otpTestEnv{
		ctx: ctx, pool: pool, store: store, hasher: codeHasher, cipher: eventCipher,
		service: service.NewChallengeService(tracer, codeHasher, eventCipher, func() {}, store, limits), limits: limits,
	}
}

func (e *otpTestEnv) newChallenge(destination string) *domain.Challenge {
	return &domain.Challenge{
		ID:          uuid.New(),
		Destination: destination,
		Channel:     domain.OtpChannelEmail,
		Purpose:     domain.CodePurposeAuthentication,
		CodeHash:    e.hasher.Hash(testCode),
		ExpiresAt:   time.Now().Add(time.Hour),
	}
}

func (e *otpTestEnv) saveChallenge(t *testing.T, challenge *domain.Challenge) {
	t.Helper()
	if _, err := e.store.ChallengeRepository().Upster(e.ctx, challenge); err != nil {
		t.Fatal(err)
	}
}

func (e *otpTestEnv) consume(id uuid.UUID, code string) error {
	_, err := e.service.Consume(e.ctx, domain.ConsumeChallengeDTO{
		ChallengeID:     id,
		Code:            code,
		ExpectedPurpose: domain.CodePurposeAuthentication,
	})
	return err
}
