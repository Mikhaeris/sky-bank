package handler

import (
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
	v1 "github.com/mikhaeris/sky-bank/proto/gen/auth/v1"
	"go.opentelemetry.io/otel/trace"
)

type AuthHandler struct {
	v1.UnimplementedAuthServer
	tracer          trace.Tracer
	identityService *service.IdentityService
	sessionsService *service.SessionsService
}

func NewAuthHandler(
	tracer trace.Tracer,
	identityService *service.IdentityService,
	sessionsService *service.SessionsService,
) *AuthHandler {
	return &AuthHandler{
		tracer:          tracer,
		identityService: identityService,
		sessionsService: sessionsService,
	}
}
