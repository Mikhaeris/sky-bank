package interceptors

import (
	"context"
	"strings"

	"github.com/mikhaeris/sky-bank/gateway/internal/pkg/jwt"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"

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

		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(identityIDMetadata, claims.Subject)
		md.Set(emailMetadata, claims.Email)
		ctx = metadata.NewOutgoingContext(ctx, md)

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
