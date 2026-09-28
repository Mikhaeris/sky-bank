package registers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	interceptor "github.com/mikhaeris/sky-bank/gateway/internal/interceptor"
	jwt "github.com/mikhaeris/sky-bank/gateway/internal/pkg/jwt"
	"github.com/mikhaeris/sky-bank/gateway/internal/pkg/ratelimit"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	customerv1 "github.com/mikhaeris/sky-bank/proto/gen/customer/v1"

	_ "google.golang.org/genproto/googleapis/rpc/errdetails"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func RegisterAll(authAddr, customerAddr string, tokenVerifier *jwt.TokenVerifier, identityLimiter *ratelimit.Limiter) (*runtime.ServeMux, authv1.AuthClient, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	mux := runtime.NewServeMux()

	authClient, authConn, err := registerAuth(ctx, mux, authAddr, tokenVerifier, identityLimiter)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	cleanup := func() {
		cancel()
		_ = authConn.Close()
	}

	err = registerCustomer(ctx, mux, customerAddr, tokenVerifier, identityLimiter)
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}

	return mux, authClient, cleanup, nil
}

func registerAuth(
	ctx context.Context,
	mux *runtime.ServeMux,
	addr string,
	tokenVerifier *jwt.TokenVerifier,
	identityLimiter *ratelimit.Limiter,
) (authv1.AuthClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		addr,
		grpcOptions(tokenVerifier, identityLimiter)...,
	)
	if err != nil {
		return nil, nil, err
	}

	authClient := authv1.NewAuthClient(conn)
	if err := authv1.RegisterAuthHandlerClient(ctx, mux, authClient); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return authClient, conn, nil
}

func registerCustomer(ctx context.Context, mux *runtime.ServeMux, addr string, tokenVerifier *jwt.TokenVerifier, identityLimiter *ratelimit.Limiter) error {
	opts := grpcOptions(tokenVerifier, identityLimiter)
	return customerv1.RegisterCustomerHandlerFromEndpoint(ctx, mux, addr, opts)
}

func grpcOptions(tokenVerifier *jwt.TokenVerifier, identityLimiter *ratelimit.Limiter) []grpc.DialOption {
	chain := []grpc.UnaryClientInterceptor{interceptor.Authenticate(tokenVerifier)}
	if identityLimiter != nil {
		chain = append(chain, interceptor.LimiterByID(identityLimiter))
	}
	return []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(chain...),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}
