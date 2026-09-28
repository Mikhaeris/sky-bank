package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mikhaeris/sky-bank/auth_service/internal/app"
	postgresclient "github.com/mikhaeris/sky-bank/auth_service/internal/clients/postgres"
	"github.com/mikhaeris/sky-bank/auth_service/internal/config"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/otp"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/mikhaeris/sky-bank/pkg/kafka"
	"github.com/mikhaeris/sky-bank/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	tracer, cleanup, err := telemetry.NewTracer(ctx, cfg.OtlpEndpoint, "auth_service")
	if err != nil {
		logger.Error("new tracer", "error", err)
		os.Exit(1)
	}
	defer cleanup()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	pool, err := postgresclient.OpenDB(&cfg.Storage)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("database connection pool established")

	store := repository.NewDataStore(tracer, pool)

	codeHash, err := otp.NewCodeHasher(cfg.OtpSecretPath)
	if err != nil {
		logger.Error("create OTP code hasher", "error", err)
		os.Exit(1)
	}

	keys, err := jwt.NewKeys(cfg.Jwt.PrivKeyPath, cfg.Jwt.AccessTokenTtl)
	if err != nil {
		logger.Error("load signing key", "error", err)
		os.Exit(1)
	}

	producer, err := kafka.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		logger.Error("create kafka producer", "error", err)
		os.Exit(1)
	}
	defer producer.Close()

	challengeService := service.NewChallengeService(
		tracer,
		codeHash,
		producer,
		store,
		cfg.OtpLimits,
	)
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		app.RunOTPCleanup(cleanupCtx, logger, challengeService)
	}()
	defer func() {
		stopCleanup()
		<-cleanupDone
	}()

	sessionsService := service.NewSessionsService(
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

	err = app.RunGRPCServer(grpcServer, cfg.Server.Grpc.Addr, logger)
	if err != nil {
		logger.Error("run grpc server", "error", err)
		os.Exit(1)
	}
}
