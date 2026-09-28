package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	oteltrace "go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

func NewOTLPExporter(ctx context.Context, otlpEndpoint string) (oteltrace.SpanExporter, error) {
	insecureOpt := otlptracehttp.WithInsecure()

	endpointOpt := otlptracehttp.WithEndpoint(otlpEndpoint)

	return otlptracehttp.New(ctx, insecureOpt, endpointOpt)
}

func NewTraceProvider(ctx context.Context, otlpEndpoint string, serviceName string) (*sdktrace.TracerProvider, error) {
	exp, err := NewOTLPExporter(ctx, otlpEndpoint)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter error: %w", err)
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	), nil
}

func NewTracer(ctx context.Context, otlpEndpoint string, serviceName string) (trace.Tracer, func(), error) {
	tp, err := NewTraceProvider(ctx, otlpEndpoint, serviceName)
	if err != nil {
		return nil, nil, err
	}
	clenup := func() { _ = tp.Shutdown(ctx) }
	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(serviceName)
	return tracer, clenup, nil
}
