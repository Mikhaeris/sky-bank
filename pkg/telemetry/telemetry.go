package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName string
	Endpoint    string
}

type Telemetry struct {
	Logger *slog.Logger
	Tracer trace.Tracer

	traces  *sdktrace.TracerProvider
	metrics *sdkmetric.MeterProvider
	logs    *sdklog.LoggerProvider
}

func Init(ctx context.Context, cfg Config) (_ *Telemetry, err error) {
	if cfg.ServiceName == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("service name and OTLP endpoint are required")
	}

	res, err := newResource(cfg.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	var traces *sdktrace.TracerProvider
	var metrics *sdkmetric.MeterProvider
	var logs *sdklog.LoggerProvider
	defer func() {
		if err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = errors.Join(err, shutdownProviders(cleanupCtx, logs, traces, metrics))
		}
	}()

	traces, err = newTraceProvider(ctx, cfg.Endpoint, res)
	if err != nil {
		return nil, fmt.Errorf("init traces: %w", err)
	}
	metrics, err = newMeterProvider(ctx, cfg.Endpoint, res)
	if err != nil {
		return nil, fmt.Errorf("init metrics: %w", err)
	}
	logs, err = newLoggerProvider(ctx, cfg.Endpoint, res)
	if err != nil {
		return nil, fmt.Errorf("init logs: %w", err)
	}
	if err = runtime.Start(runtime.WithMeterProvider(metrics)); err != nil {
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	otel.SetTracerProvider(traces)
	otel.SetMeterProvider(metrics)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Telemetry{
		Logger:  otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(logs)),
		Tracer:  traces.Tracer(cfg.ServiceName),
		traces:  traces,
		metrics: metrics,
		logs:    logs,
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return shutdownProviders(ctx, t.logs, t.traces, t.metrics)
}

func shutdownProviders(ctx context.Context, logs *sdklog.LoggerProvider, traces *sdktrace.TracerProvider, metrics *sdkmetric.MeterProvider) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	shutdown := func(name string, stop func(context.Context) error) {
		wg.Go(func() {
			if err := stop(ctx); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("shutdown %s: %w", name, err))
				mu.Unlock()
			}
		})
	}
	if logs != nil {
		shutdown("logs", logs.Shutdown)
	}
	if traces != nil {
		shutdown("traces", traces.Shutdown)
	}
	if metrics != nil {
		shutdown("metrics", metrics.Shutdown)
	}
	wg.Wait()
	return errors.Join(errs...)
}

func newResource(serviceName string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName)),
	)
}
