package unit_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryInterceptorKeepsServerUsableAfterPanic(t *testing.T) {
	var output bytes.Buffer
	interceptor := interceptors.RecoveryInterceptors(slog.New(slog.NewTextHandler(&output, nil)))
	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.Auth/CompleteAuthentication"}

	response, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		panic("unexpected handler failure")
	})
	if response != nil || status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal error" {
		t.Fatalf("panic response = %v, error = %v", response, err)
	}
	for _, part := range []string{info.FullMethod, "unexpected handler failure", "stack"} {
		if !strings.Contains(output.String(), part) {
			t.Fatalf("panic log lacks %q: %s", part, output.String())
		}
	}

	response, err = interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return "ready", nil
	})
	if err != nil || response != "ready" {
		t.Fatalf("next request response = %v, error = %v", response, err)
	}
}
