package service

import (
	"log/slog"

	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/utils"
)

type AuthService struct {
	jwtKey   *utils.Keys
	logger   *slog.Logger
	userRepo *repository.UserRepository
}

func NewAuthService(jwtKey *utils.Keys, logger *slog.Logger, userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		jwtKey:   jwtKey,
		logger:   logger,
		userRepo: userRepo,
	}
}
