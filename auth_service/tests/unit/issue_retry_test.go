package unit_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIssueRetryDetails(t *testing.T) {
	retryAt := time.Now().UTC().Add(time.Minute)
	_, err := interceptors.ErrorInterceptor(slog.New(slog.NewTextHandler(io.Discard, nil)))(
		context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth/start"},
		func(context.Context, any) (any, error) {
			return nil, fmt.Errorf("issue challenge: %w", &service.IssueRateLimitError{AvailableAt: retryAt})
		})
	s := status.Convert(err)
	if s.Code() != codes.ResourceExhausted {
		t.Fatalf("status=%v", s)
	}
	var reasonFound, retryFound bool
	for _, detail := range s.Details() {
		switch d := detail.(type) {
		case *errdetails.ErrorInfo:
			reasonFound = d.Reason == "OTP_RATE_LIMITED" && d.Metadata["retry_at"] == retryAt.Format(time.RFC3339Nano)
		case *errdetails.RetryInfo:
			retryFound = d.RetryDelay.AsDuration() > 0 && d.RetryDelay.AsDuration() <= time.Minute
		}
	}
	if !reasonFound || !retryFound {
		t.Fatalf("missing retry details: %v", s.Details())
	}
}
