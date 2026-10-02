package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	oteltrace "go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func NewOTLPExporter(ctx context.Context, otlpEndpoint string) (oteltrace.SpanExporter, error) {
	insecureOpt := otlptracehttp.WithInsecure()

	endpointOpt := otlptracehttp.WithEndpoint(otlpEndpoint)

	return otlptracehttp.New(ctx, insecureOpt, endpointOpt)
}

func NewTraceProvider(ctx context.Context, otlpEndpoint string, serviceName string) (*sdktrace.TracerProvider, error) {
	res, err := newResource(serviceName)
	if err != nil {
		return nil, fmt.Errorf("create trace resource: %w", err)
	}
	return newTraceProvider(ctx, otlpEndpoint, res)
}

func newTraceProvider(ctx context.Context, endpoint string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := NewOTLPExporter(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	), nil
}

func NewTracer(ctx context.Context, otlpEndpoint string, serviceName string) (trace.Tracer, func(), error) {
	tp, err := NewTraceProvider(ctx, otlpEndpoint, serviceName)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutdownCtx)
	}
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(serviceName)
	return tracer, cleanup, nil
}
