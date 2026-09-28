package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/mikhaeris/sky-bank/gateway/internal/config"
	"github.com/mikhaeris/sky-bank/gateway/internal/middleware"
	jwt "github.com/mikhaeris/sky-bank/gateway/internal/pkg/jwt"
	"github.com/mikhaeris/sky-bank/gateway/internal/pkg/ratelimit"
	"github.com/mikhaeris/sky-bank/gateway/internal/registers"
	routes "github.com/mikhaeris/sky-bank/gateway/internal/route"
	"github.com/mikhaeris/sky-bank/pkg/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config.GetConfig(logger)

	_, cleanup, err := telemetry.NewTracer(ctx, cfg.OtlpEndpoint, "api_gateway")
	if err != nil {
		logger.Error("new tracer", "error", err)
		os.Exit(1)
	}
	defer cleanup()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tokenVerifier, err := jwt.NewTokenVerifier(cfg.Jwt.PubKeyPath)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	var publicLimiter, identityLimiter *ratelimit.Limiter
	publicMiddleware := func(next http.Handler) http.Handler { return next }
	if cfg.RateLimiter.Enabled {
		publicLimiter, err = ratelimit.New(cfg.RateLimiter.PublicRPS, cfg.RateLimiter.PublicBurst)
		if err != nil {
			logger.Error("invalid public rate limiter", "error", err)
			os.Exit(1)
		}
		identityLimiter, err = ratelimit.New(cfg.RateLimiter.PrivateRPS, cfg.RateLimiter.PrivateBurst)
		if err != nil {
			logger.Error("invalid private rate limiter", "error", err)
			os.Exit(1)
		}
		publicMiddleware = middleware.RateLimit(publicLimiter)
	}

	ctx, cancelLimiters := context.WithCancel(context.Background())
	defer cancelLimiters()
	if publicLimiter != nil {
		go publicLimiter.RunCleanup(ctx)
		go identityLimiter.RunCleanup(ctx)
	}

	serverMux, authClient, cleanup, err := registers.RegisterAll(cfg.Auth.Addr, cfg.Customer.Addr, tokenVerifier, identityLimiter)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer cleanup()

	mux := http.NewServeMux()
	routes.RegisterAuth(mux, authClient, serverMux, publicMiddleware)

	mux.Handle(
		"/api/v1/",
		http.StripPrefix("/api/v1", serverMux),
	)

	err = http.ListenAndServe(cfg.Rest.Addr, otelhttp.NewHandler(mux, "gateway.http"))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
