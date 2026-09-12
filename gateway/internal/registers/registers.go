package registers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authv1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/gateway/internal/interceptors"
	"github.com/mikhaeris/sky-bank/gateway/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RegisterAll(authAddr string, tokenVerifier *utils.TokenVerifier) (*runtime.ServeMux, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	cleanup := func() {
		cancel()
	}

	mux := runtime.NewServeMux()

	err := registerAuth(ctx, mux, authAddr, tokenVerifier)
	if err != nil {
		return nil, cleanup, err
	}

	return mux, cleanup, nil
}

func registerAuth(ctx context.Context, mux *runtime.ServeMux, addr string, tokenVerifier *utils.TokenVerifier) error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptors.Authenticate(tokenVerifier)),
	}
	return authv1.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, addr, opts)
}
