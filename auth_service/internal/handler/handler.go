package handler

import (
	"log/slog"

	"github.com/mikhaeris/bank-test/auth_service/internal/service"
)

type AuthHandler struct {
	logger      *slog.Logger
	authService *service.AuthService
}

func NewAuthHandler(logger *slog.Logger, authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		logger:      logger,
		authService: authService,
	}
}
