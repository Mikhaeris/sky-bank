package telemetry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/pkg/telemetry"
	"go.opentelemetry.io/otel"
)

func TestInitExportsAllSignalsOnShutdown(t *testing.T) {
	previousTraces := otel.GetTracerProvider()
	previousMetrics := otel.GetMeterProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTraces)
		otel.SetMeterProvider(previousMetrics)
		otel.SetTextMapPropagator(previousPropagator)
	})

	var mu sync.Mutex
	received := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received[r.URL.Path]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	startupCtx, cancelStartup := context.WithCancel(context.Background())
	observer, err := telemetry.Init(startupCtx, telemetry.Config{
		ServiceName: "telemetry_test",
		Endpoint:    strings.TrimPrefix(server.URL, "http://"),
	})
	if err != nil {
		t.Fatalf("initialize telemetry: %v", err)
	}
	shutdownDone := false
	defer func() {
		if shutdownDone {
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = observer.Shutdown(shutdownCtx)
	}()

	ctx, span := observer.Tracer.Start(context.Background(), "test-operation")
	observer.Logger.InfoContext(ctx, "test log")
	span.End()

	counter, err := otel.Meter("telemetry_test").Int64Counter("test.requests")
	if err != nil {
		t.Fatalf("create metric: %v", err)
	}
	counter.Add(context.Background(), 1)
	cancelStartup()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := observer.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown telemetry: %v", err)
	}
	shutdownDone = true

	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{"/v1/traces", "/v1/metrics", "/v1/logs"} {
		if received[path] == 0 {
			t.Errorf("no OTLP request received at %s", path)
		}
	}
}
