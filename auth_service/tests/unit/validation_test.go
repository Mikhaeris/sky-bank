package unit_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/mikhaeris/sky-bank/auth_service/internal/handler"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server"
	"github.com/mikhaeris/sky-bank/auth_service/internal/server/interceptors"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestValidationInterceptor(t *testing.T) {
	tests := []struct {
		name      string
		request   any
		valid     bool
		checkBody func(*testing.T, any)
	}{
		{"normalizes login email", &authv1.StartAuthenticationRequest{Email: "  Alice@Example.COM  "}, true,
			func(t *testing.T, req any) {
				if got := req.(*authv1.StartAuthenticationRequest).Email; got != "alice@example.com" {
					t.Fatalf("email = %q", got)
				}
			}},
		{"rejects invalid login email", &authv1.StartAuthenticationRequest{Email: "invalid"}, false, nil},
		{"rejects malformed OTP", &authv1.CompleteAuthenticationRequest{
			ChallengeId: "550e8400-e29b-41d4-a716-446655440000", OtpCode: "12AB56"}, false, nil},
		{"rejects email challenge with invalid destination", &authv1.CreateChallengeRequest{
			Destination: "invalid", Channel: authv1.OtpChannel_OTP_CHANNEL_EMAIL,
			Purpose: authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION}, false, nil},
		{"trims SMS destination", &authv1.CreateChallengeRequest{
			Destination: "  +123456789  ", Channel: authv1.OtpChannel_OTP_CHANNEL_SMS,
			Purpose: authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION}, true,
			func(t *testing.T, req any) {
				if got := req.(*authv1.CreateChallengeRequest).Destination; got != "+123456789" {
					t.Fatalf("destination = %q", got)
				}
			}},
		{"rejects blank SMS destination", &authv1.CreateChallengeRequest{
			Destination: "   ", Channel: authv1.OtpChannel_OTP_CHANNEL_SMS,
			Purpose: authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION}, false, nil},
		{"normalizes verification destination", &authv1.VerifyChallengeRequest{
			ChallengeId: "550e8400-e29b-41d4-a716-446655440000", Code: "012345",
			Destination: "  Alice@Example.COM  ", Purpose: authv1.OtpPurpose_OTP_PURPOSE_EMAIL_VERIFICATION}, true,
			func(t *testing.T, req any) {
				if got := req.(*authv1.VerifyChallengeRequest).Destination; got != "alice@example.com" {
					t.Fatalf("destination = %q", got)
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			_, err := interceptors.ValidationInterceptor()(context.Background(), tt.request,
				&grpc.UnaryServerInfo{FullMethod: "/api.auth.v1.Auth/Test"},
				func(_ context.Context, request any) (any, error) {
					called = true
					if tt.checkBody != nil {
						tt.checkBody(t, request)
					}
					return nil, nil
				})
			if tt.valid {
				if err != nil || !called {
					t.Fatalf("valid request rejected: called=%v err=%v", called, err)
				}
			} else if err == nil || called {
				t.Fatalf("invalid request accepted: called=%v err=%v", called, err)
			}
		})
	}
}

func TestValidationErrorIsPublicInvalidArgument(t *testing.T) {
	invalid := &authv1.StartAuthenticationRequest{Email: "invalid"}
	_, err := interceptors.ErrorInterceptor(nil)(context.Background(), invalid,
		&grpc.UnaryServerInfo{FullMethod: "/api.auth.v1.Auth/StartAuthentication"},
		func(ctx context.Context, req any) (any, error) {
			return interceptors.ValidationInterceptor()(ctx, req, nil, func(context.Context, any) (any, error) {
				t.Fatal("invalid request reached handler")
				return nil, nil
			})
		})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("validation error = %v", err)
	}
}

func TestAuthServerRunsValidationBeforeHandler(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	grpcServer := server.NewGRPCServer(
		handler.NewAuthHandler(noop.NewTracerProvider().Tracer("test"), nil, nil),
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
	_, err = authv1.NewAuthClient(conn).StartAuthentication(ctx, &authv1.StartAuthenticationRequest{Email: "invalid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid email reached handler: %v", err)
	}
	_, err = authv1.NewAuthClient(conn).CompleteAuthentication(ctx, &authv1.CompleteAuthenticationRequest{ChallengeId: "not-a-uuid"})
	if status.Code(err) != codes.InvalidArgument || status.Convert(err).Message() != "invalid id" {
		t.Fatalf("invalid UUID response changed: %v", err)
	}
}
