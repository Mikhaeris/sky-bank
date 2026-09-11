package service

import (
	"context"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"golang.org/x/crypto/bcrypt"
)

func (a *AuthService) RegisterUser(ctx context.Context, dto domain.UserDTO) (uuid.UUID, error) {
	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return uuid.Nil(), err
	}

	// generate uuid
	userUUID := uuid.New()

	// dto to User
	user := domain.NewUser(dto, userUUID, string(hash))

	// Validate

	// insert to auth_db
	err = a.userRepo.Insert(ctx, user)
	if err != nil {
		return uuid.Nil(), err
	}

	// send user_service information to create

	// generate token for email
	token, err := a.NewToken(ctx, userUUID, 3*24*time.Hour, domain.ScopeActivation)
	if err != nil {
		return uuid.Nil(), err
	}

	// send confirmation email
	go func(user domain.User, activationToken string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.WelcomeMessageRequest{
			IdentityUuid:    user.ID.String(),
			Email:           user.Email,
			ActivationToken: activationToken,
		}

		if _, err := a.notificationClient.SendWelcomeMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send welcome message",
				"error", err,
				"user_id", user.ID,
			)
		}
	}(*user, token.Plaintext)

	return user.ID, nil
}

func (a *AuthService) ActivateUser(ctx context.Context, dto domain.ActivateUserDTO) (domain.User, error) {
	return domain.User{}, nil
}
