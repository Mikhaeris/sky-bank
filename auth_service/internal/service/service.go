package service

import (
	"log/slog"

	"github.com/mikhaeris/bank-test/auth_service/internal/repository"
)

type AuthService struct {
	logger   *slog.Logger
	userRepo *repository.UserRepository
}

func NewAuthService(logger *slog.Logger, userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		logger:   logger,
		userRepo: userRepo,
	}
}
