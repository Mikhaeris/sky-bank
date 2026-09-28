package routes

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/mikhaeris/sky-bank/gateway/internal/handler"
	authv1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
)

func RegisterAuth(mux *http.ServeMux, authClient authv1.AuthClient, grpcGateway *runtime.ServeMux, publicMiddleware func(http.Handler) http.Handler) {
	authHandler := &handler.AuthHandler{Auth: authClient, Mux: grpcGateway}
	mux.Handle("POST /api/v1/auth/start", publicMiddleware(http.StripPrefix("/api/v1", grpcGateway)))
	mux.Handle("POST /api/v1/auth/complete", publicMiddleware(http.HandlerFunc(authHandler.Complete)))
	mux.Handle("POST /api/v1/auth/refresh", publicMiddleware(http.HandlerFunc(authHandler.Refresh)))
}
