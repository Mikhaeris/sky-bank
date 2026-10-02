package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestOTPOutboxClaimRejectsInvalidArguments(t *testing.T) {
	repo := repository.NewOTPOutboxRepository(noop.NewTracerProvider().Tracer("test"), nil)
	for _, args := range []struct {
		limit int
		lease time.Duration
	}{{0, time.Second}, {-1, time.Second}, {1, 0}, {1, -time.Second}, {1, time.Nanosecond}} {
		if _, err := repo.Claim(context.Background(), args.limit, args.lease); err == nil {
			t.Fatalf("accepted invalid claim: %+v", args)
		}
	}
}
