package interceptors

import (
	"context"
	"strings"

	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/gateway/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func Authenticate(
	verifier *utils.TokenVerifier,
) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if !requiresAuthentication(method) {
			return invoker(
				ctx,
				method,
				req,
				reply,
				cc,
				opts...,
			)
		}

		token, err := utils.ExtractBearerToken(ctx)
		if err != nil {
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

		md.Delete(utils.UserIDMetadata)

		md.Delete("authorization")

		md.Set(
			utils.UserIDMetadata,
			claims.Subject,
		)

		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(
			ctx,
			method,
			req,
			reply,
			cc,
			opts...,
		)
	}
}

func requiresAuthentication(method string) bool {
	policy, ok := getAuthPolicy(method)

	if !ok {
		return true
	}

	return policy.Authentication !=
		authv1.Authentication_AUTHENTICATION_PUBLIC
}

func getAuthPolicy(method string) (*authv1.AuthPolicy, bool) {
	fullName := strings.TrimPrefix(method, "/")
	fullName = strings.ReplaceAll(fullName, "/", ".")

	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(
		protoreflect.FullName(fullName),
	)
	if err != nil {
		return nil, false
	}

	methodDescriptor, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, false
	}

	options, ok := methodDescriptor.Options().(*descriptorpb.MethodOptions)
	if !ok {
		return nil, false
	}

	if !proto.HasExtension(options, authv1.E_Auth) {
		return nil, false
	}

	value := proto.GetExtension(options, authv1.E_Auth)

	policy, ok := value.(*authv1.AuthPolicy)
	if !ok || policy == nil {
		return nil, false
	}

	return policy, true
}
