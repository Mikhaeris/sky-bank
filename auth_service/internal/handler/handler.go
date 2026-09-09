package handler

import (
	"log/slog"

	v1 "github.com/mikhaeris/sky-bank/auth_service/api/auth/v1"
	"github.com/mikhaeris/sky-bank/auth_service/internal/service"
)

type AuthHandler struct {
	v1.UnimplementedAuthSericeServer
	logger      *slog.Logger
	authService *service.AuthService
}

func NewAuthHandler(logger *slog.Logger, authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		logger:      logger,
		authService: authService,
	}
}
