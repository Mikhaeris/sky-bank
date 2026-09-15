package interceptors

import (
	"context"
	"strings"

	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	jwt "github.com/mikhaeris/sky-bank/gateway/internal/lib"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	identityIDMetadata = "x-identity-id"
	emailMetadata      = "x-email"
)

var accessTokenExemptMethods = map[string]struct{}{
	authv1.Auth_StartAuthentication_FullMethodName:    {},
	authv1.Auth_CompleteAuthentication_FullMethodName: {},
	authv1.Auth_RefreshTokens_FullMethodName:          {},
}

func Authenticate(verifier *jwt.TokenVerifier) grpc.UnaryClientInterceptor {
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

		token, ok := extractBearerToken(ctx)
		if !ok {
			return status.Error(
				codes.Unauthenticated,
				"access token required",
			)
		}

		claims, err := verifier.Verify(token)
		if err != nil {
			return status.Error(
				codes.Unauthenticated,
				"invalid access token",
			)
		}

		ctx = metadata.AppendToOutgoingContext(
			ctx,
			identityIDMetadata, claims.Subject,
			emailMetadata, claims.Email,
		)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func extractBearerToken(ctx context.Context) (string, bool) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", false
	}

	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], true
}
