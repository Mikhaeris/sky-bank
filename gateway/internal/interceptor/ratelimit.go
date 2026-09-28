package interceptors

import (
	"context"

	"github.com/mikhaeris/sky-bank/gateway/internal/pkg/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func LimiterByID(limiter *ratelimit.Limiter) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if _, exempt := accessTokenExemptMethods[method]; exempt {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "identity required")
		}
		identities := md.Get(identityIDMetadata)
		if len(identities) != 1 || identities[0] == "" {
			return status.Error(codes.Unauthenticated, "identity required")
		}
		if !limiter.Allow(identities[0]) {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
