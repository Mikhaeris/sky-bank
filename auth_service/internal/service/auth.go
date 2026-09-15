package service

import (
	"log/slog"

	otpprovider "github.com/mikhaeris/sky-bank/auth_service/internal/clients/otpProvider"
	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/lib/jwt"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
)

type AuthService struct {
	jwtKey             *jwt.Keys
	logger             *slog.Logger
	codeHash           *domain.CodeHasher
	otpProvider        otpprovider.OtpProvider
	notificationClient notificationv1.NotificationServiceClient
	otpRepo            *repository.OtpRepository
	sessionRepo        *repository.SessionRepository
	identityRepo       *repository.IdentityRepository
}

func NewAuthService(
	jwtKey *jwt.Keys,
	logger *slog.Logger,
	codeHash *domain.CodeHasher,
	otpProvider otpprovider.OtpProvider,
	notificationClient notificationv1.NotificationServiceClient,
	otpRepo *repository.OtpRepository,
	sessionRepo *repository.SessionRepository,
	identiRepo *repository.IdentityRepository,
) *AuthService {
	return &AuthService{
		jwtKey:             jwtKey,
		logger:             logger,
		codeHash:           codeHash,
		otpProvider:        otpProvider,
		notificationClient: notificationClient,
		otpRepo:            otpRepo,
		sessionRepo:        sessionRepo,
		identityRepo:       identiRepo,
	}
}
