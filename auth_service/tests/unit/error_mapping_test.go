package unit_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	"github.com/mikhaeris/sky-bank/pkg/apperr"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestErrorInterceptorMapsExpectedFailures(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    codes.Code
		message string
		reason  string
	}{
		{"invalid id", handler.ErrInvalidID, codes.InvalidArgument, "invalid id", "INVALID_ID"},
		{"invalid channel", handler.ErrInvalidOTPChannel, codes.InvalidArgument, "invalid otp channel", "INVALID_OTP_CHANNEL"},
		{"invalid purpose enum", handler.ErrInvalidOTPPurpose, codes.InvalidArgument, "invalid otp purpose", "INVALID_OTP_PURPOSE"},
		{"invalid otp", service.ErrOtpCodeInvalid, codes.InvalidArgument, "invalid otp code", "INVALID_OTP_CODE"},
		{"wrong destination", service.ErrDestinationInvalid, codes.InvalidArgument, "invalid destination", "INVALID_DESTINATION"},
		{"wrong purpose", service.ErrPurposeInvalid, codes.InvalidArgument, "invalid purpose", "INVALID_PURPOSE"},
		{"otp rate limit", service.ErrOtpRateLimited, codes.ResourceExhausted, "otp rate limit exceeded", "OTP_RATE_LIMITED"},
		{"invalid refresh token", service.ErrInvalidRefreshToken, codes.Unauthenticated, "invalid refresh token", "INVALID_REFRESH_TOKEN"},
		{"authentication required", apperr.New(apperr.Unauthenticated, "AUTHENTICATION_REQUIRED", "authentication required"), codes.Unauthenticated, "authentication required", "AUTHENTICATION_REQUIRED"},
		{"session not found", service.ErrSessionNotFound, codes.NotFound, "session not found", "SESSION_NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&output, nil))
			_, err := interceptors.ErrorInterceptor(logger)(
				context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/auth.v1.Auth/Test"},
				func(context.Context, any) (any, error) { return nil, fmt.Errorf("operation: %w", tt.err) },
			)
			if status.Code(err) != tt.code || status.Convert(err).Message() != tt.message {
				t.Fatalf("got %v, want %v: %s", err, tt.code, tt.message)
			}
			details := status.Convert(err).Details()
			if len(details) != 1 {
				t.Fatalf("error details = %v, want one ErrorInfo", details)
			}
			info, ok := details[0].(*errdetails.ErrorInfo)
			if !ok || info.Reason != tt.reason || info.Domain != "auth_service" {
				t.Fatalf("error details = %v, want reason %q", details, tt.reason)
			}
			if output.Len() != 0 {
				t.Fatalf("expected failure was logged as a server error: %q", output.String())
			}
		})
	}
}

func TestErrorInterceptorHidesUnexpectedFailureAndHandlesContext(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.Auth/Test"}
	cause := errors.New("database connection refused")
	_, err := interceptors.ErrorInterceptor(logger)(context.Background(), nil, info,
		func(context.Context, any) (any, error) { return nil, fmt.Errorf("list sessions: %w", cause) })
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal error" {
		t.Fatalf("unexpected public error: %v", err)
	}
	if !strings.Contains(output.String(), cause.Error()) || !strings.Contains(output.String(), info.FullMethod) {
		t.Fatalf("log lacks operation context or cause: %q", output.String())
	}

	output.Reset()
	_, err = interceptors.ErrorInterceptor(logger)(context.Background(), nil, info,
		func(context.Context, any) (any, error) {
			return nil, fmt.Errorf("secret detail: %w", apperr.New(0, "BAD_KIND", "unsafe message"))
		})
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal error" {
		t.Fatalf("unknown public error kind leaked to client: %v", err)
	}
	if !strings.Contains(output.String(), "secret detail") {
		t.Fatalf("unknown error missing from log: %q", output.String())
	}

	output.Reset()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = interceptors.ErrorInterceptor(logger)(ctx, nil, info,
		func(context.Context, any) (any, error) { return nil, fmt.Errorf("query: %w", context.Canceled) })
	if status.Code(err) != codes.Canceled || output.Len() != 0 {
		t.Fatalf("canceled request: error=%v log=%q", err, output.String())
	}
	_, err = interceptors.ErrorInterceptor(logger)(ctx, nil, info,
		func(context.Context, any) (any, error) { return nil, cause })
	if status.Code(err) != codes.Canceled || output.Len() != 0 {
		t.Fatalf("canceled request with driver error: error=%v log=%q", err, output.String())
	}

	_, err = interceptors.ErrorInterceptor(logger)(context.Background(), nil, info,
		func(context.Context, any) (any, error) { return nil, fmt.Errorf("query: %w", context.DeadlineExceeded) })
	if status.Code(err) != codes.DeadlineExceeded || status.Convert(err).Message() != context.DeadlineExceeded.Error() || !strings.Contains(output.String(), "query: context deadline exceeded") {
		t.Fatalf("internal timeout: error=%v log=%q", err, output.String())
	}
}

func TestRevokeMissingSessionReturnsScenarioError(t *testing.T) {
	identityID := uuid.New()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-identity-id", identityID.String(), "x-email", "user@example.invalid",
	))
	svc := service.NewSessionsService(noop.NewTracerProvider().Tracer("test"), refreshStore{db: refreshDB{queryErr: errors.New("unexpected query")}})
	err := svc.RevokeSession(ctx, domain.RevokeSessionDTO{SessionID: uuid.New()})
	if !errors.Is(err, service.ErrSessionNotFound) {
		t.Fatalf("missing session = %v, want ErrSessionNotFound", err)
	}
}

func TestAuthGRPCServerInstallsErrorInterceptor(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")
	grpcServer := server.NewGRPCServer(
		handler.NewAuthHandler(tracer, nil, service.NewSessionsService(tracer, nil)),
		handler.NewChallengeHandler(nil), logger,
	)
	serveDone := make(chan error, 1)
	go func() { serveDone <- grpcServer.Serve(listener) }()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		<-serveDone
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := authv1.NewAuthClient(conn)

	_, err = client.CompleteAuthentication(ctx, &authv1.CompleteAuthenticationRequest{ChallengeId: "not-a-uuid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid challenge ID: %v", err)
	}
	if details := status.Convert(err).Details(); len(details) != 1 {
		t.Fatalf("invalid challenge ID details: %v", details)
	} else if info, ok := details[0].(*errdetails.ErrorInfo); !ok || info.Reason != "INVALID_ID" {
		t.Fatalf("invalid challenge ID reason: %v", details)
	}
	_, err = client.GetSessions(ctx, &emptypb.Empty{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing authentication metadata: %v", err)
	}
}
