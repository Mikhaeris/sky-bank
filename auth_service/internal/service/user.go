package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/repository"
	notificationv1 "github.com/mikhaeris/sky-bank/notification_service/api/notification/v1"
	"golang.org/x/crypto/bcrypt"
)

func (a *AuthService) RegisterUser(ctx context.Context, dto domain.UserDTO) (uuid.UUID, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return uuid.Nil(), err
	}

	userUUID := uuid.New()

	user := domain.NewUser(dto, userUUID, string(hash))

	// Validate

	err = a.identiRepo.Insert(ctx, user)
	if err != nil {
		return uuid.Nil(), err
	}

	// send user_service information to create

	token, err := a.NewToken(ctx, user.Id, 3*24*time.Hour, domain.ScopeActivation)
	if err != nil {
		return uuid.Nil(), err
	}

	go func(user domain.User, activationToken string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.WelcomeMessageRequest{
			IdentityUuid:    user.Id.String(),
			Email:           user.Email,
			ActivationToken: activationToken,
		}

		if _, err := a.notificationClient.SendWelcomeMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send welcome message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user, token.Plaintext)

	return user.Id, nil
}

func (a *AuthService) ActivateUser(ctx context.Context, dto domain.ActivateUserDTO) (*domain.User, error) {
	user, err := a.identiRepo.GetForToken(ctx, domain.ScopeActivation, dto.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, fmt.Errorf("invalid or expired activation token")
		default:
			return nil, fmt.Errorf("server internal error")
		}
	}

	user.Activated = true

	err = a.identiRepo.Update(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, fmt.Errorf("unable to update the record due to an edit conflict, please try again")
		default:
			return nil, fmt.Errorf("server internal error")
		}
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeActivation, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	// kafka
	go func(user domain.User) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.WelcomeActivatedMessageRequest{
			Email: user.Email,
		}

		if _, err := a.notificationClient.SendWelcomeActivatedMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send welcome message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user)

	return user, nil
}

func (a *AuthService) ResetPassword(ctx context.Context, dto domain.ResetPasswordDTO) (*domain.User, error) {
	user, err := a.identiRepo.GetForToken(ctx, domain.ScopeResetPassword, dto.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, fmt.Errorf("invalid or expired activation token")
		default:
			return nil, fmt.Errorf("server internal error")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	user.PasswordHash = string(hash)

	err = a.identiRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeResetPassword, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	go func(user domain.User) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.ConfirmPasswordResetMessageRequest{
			Email: user.Email,
		}

		if _, err := a.notificationClient.SendConfirmPasswordResetMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send confirm password reset message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user)

	return user, nil
}

func (a *AuthService) RecoverCompromisedAccount(ctx context.Context, dto domain.ResetPasswordDTO) (*domain.User, error) {
	user, err := a.identiRepo.GetForToken(ctx, domain.ScopeSecurity, dto.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, fmt.Errorf("invalid or expired activation token")
		default:
			return nil, fmt.Errorf("server internal error")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	user.PasswordHash = string(hash)

	err = a.identiRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeSecurity, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeResetPassword, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeAuthentication, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	go func(user domain.User) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.ConfirmPasswordResetMessageRequest{
			Email: user.Email,
		}

		if _, err := a.notificationClient.SendConfirmPasswordResetMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send confirm password reset message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user)

	return user, nil
}
