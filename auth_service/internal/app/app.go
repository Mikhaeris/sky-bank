package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	w "github.com/mikhaeris/sky-bank/auth_service/internal/app/workers"
	postgresclient "github.com/mikhaeris/sky-bank/auth_service/internal/clients/postgres"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/hasher"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/mikhaeris/sky-bank/pkg/kafka"
	"github.com/mikhaeris/sky-bank/pkg/telemetry"
)

func Run(logger *slog.Logger) (runErr error) {
	bootstrapLogger := logger
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	cfg := config.GetConfig(logger)

	observability, err := telemetry.Init(
		context.Background(),
		telemetry.Config{ServiceName: "auth_service", Endpoint: cfg.OtlpEndpoint},
	)
	if err != nil {
		return fmt.Errorf("init telemetry: %w", err)
	}
	logger = observability.Logger
	defer func() {
		if runErr != nil {
			logger.Error("run auth service", "error", runErr)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := observability.Shutdown(shutdownCtx); err != nil {
			bootstrapLogger.Error("shutdown telemetry", "error", err)
		}
	}()
	tracer := observability.Tracer

	pool, err := postgresclient.OpenDB(&cfg.Storage)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	logger.Info("database connection pool established")

	store := repository.NewDataStore(tracer, pool)

	codeHash, err := hasher.NewCodeHasher(cfg.OtpSecretPath)
	if err != nil {
		return fmt.Errorf("create OTP code hasher: %w", err)
	}

	outboxCipher, err := hasher.NewEventCipher(cfg.OtpOutboxSecretPath)
	if err != nil {
		return fmt.Errorf("create OTP outbox cipher: %w", err)
	}

	keys, err := jwt.NewKeys(cfg.Jwt.PrivKeyPath, cfg.Jwt.AccessTokenTtl)
	if err != nil {
		return fmt.Errorf("load signing key: %w", err)
	}

	producer, err := kafka.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		return fmt.Errorf("create kafka producer: %w", err)
	}
	defer producer.Close()

	relay := w.NewOTPOutboxRelay(logger, tracer, store.OTPOutboxRepository(), outboxCipher, producer)
	challengeService := service.NewChallengeService(
		tracer,
		codeHash,
		outboxCipher,
		relay.Wake,
		store,
		cfg.OtpLimits,
	)

	sessionsService := service.NewSessionsService(
		tracer,
		store,
	)

	identityService := service.NewIdentityService(
		tracer,
		keys,
		producer,
		challengeService,
		store,
	)

	authHandler := handler.NewAuthHandler(tracer, identityService, sessionsService)
	challengeHandler := handler.NewChallengeHandler(challengeService)

	grpcServer := server.NewGRPCServer(authHandler, challengeHandler, logger)

	workersCtx, stopWorkers := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	workers.Go(func() { relay.Run(workersCtx) })
	workers.Go(func() { w.RunOTPCleanup(workersCtx, logger, challengeService) })
	defer func() {
		stopWorkers()
		workers.Wait()
	}()

	if err := RunGRPCServer(ctx, grpcServer, cfg.Server.Grpc.Addr, logger); err != nil {
		return fmt.Errorf("run grpc server: %w", err)
	}
	return nil
}
