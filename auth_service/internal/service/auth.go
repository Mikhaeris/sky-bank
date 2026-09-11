package service

import (
	"log/slog"

	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	"github.com/mikhaeris/sky-bank/auth_service/internal/utils"
	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
)

type AuthService struct {
	jwtKey             *utils.Keys
	logger             *slog.Logger
	notificationClient notificationv1.NotificationServiceClient
	tokenRepo          *repository.TokenRepository
	userRepo           *repository.UserRepository
}

func NewAuthService(jwtKey *utils.Keys, logger *slog.Logger, notificationClient notificationv1.NotificationServiceClient, tokenRepo *repository.TokenRepository, userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		jwtKey:             jwtKey,
		logger:             logger,
		notificationClient: notificationClient,
		tokenRepo:          tokenRepo,
		userRepo:           userRepo,
	}
}
