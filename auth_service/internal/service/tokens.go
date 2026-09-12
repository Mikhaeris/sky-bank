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

func (a *AuthService) NewToken(ctx context.Context, identityId uuid.UUID, ttl time.Duration, scope string) (*domain.Token, error) {
	token := domain.GenerateToken(identityId, ttl, scope)

	err := a.tokenRepo.Insert(ctx, token)
	return token, err
}

func (a *AuthService) CreateTokenPair(ctx context.Context, user *domain.User) (*domain.TokensDTO, error) {
	var dtoRes domain.TokensDTO

	accessToken, err := a.jwtKey.CreateToken(user)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
	}
	dtoRes.Access = accessToken

	refreshToken, err := a.NewToken(ctx, user.Id, 24*time.Hour, domain.ScopeAuthentication)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
	}
	dtoRes.Refresh = refreshToken.Plaintext

	return &dtoRes, nil
}

func (a *AuthService) CreateAuthenticationToken(ctx context.Context, dto domain.UserDTO) (*domain.TokensDTO, error) {

	// validate

	user, err := a.identiRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return nil, err
	}

	if user.Activated == false {
		return nil, fmt.Errorf("user not activated")
	}

	// check passwords
	match, err := domain.Matches(user.PasswordHash, dto.Password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, fmt.Errorf("user data is incorrect")
	}

	dtoRes, err := a.CreateTokenPair(ctx, user)

	// send email: new log in to account if its not you continue to this link
	securityToken, err := a.NewToken(ctx, user.Id, 30*time.Minute, domain.ScopeSecurity)
	go func(user domain.User, securityToken string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.RecoveryMessageRequest{
			IdentityUuid:  user.Id.String(),
			Email:         user.Email,
			SecurityToken: securityToken,
		}

		if _, err := a.notificationClient.SendRecoveryMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send recovery message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user, securityToken.Plaintext)

	return dtoRes, nil
}

func (a *AuthService) RefreshTokens(ctx context.Context, dto domain.TokensDTO) (*domain.TokensDTO, error) {
	user, err := a.identiRepo.GetForToken(ctx, domain.ScopeAuthentication, dto.Refresh)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, fmt.Errorf("invalid or expired refresh token")
		default:
			return nil, fmt.Errorf("server internal error")
		}
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeAuthentication, user.Id)
	if err != nil {
		return nil, fmt.Errorf("server internal error")
	}

	dtoRes, err := a.CreateTokenPair(ctx, user)

	return dtoRes, nil
}

func (a *AuthService) CreateActivationToken(ctx context.Context, dto domain.UserDTO) (*domain.User, error) {
	user, err := a.identiRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return nil, fmt.Errorf("user not exit")
	}

	if user.Activated {
		return nil, fmt.Errorf("internal server error")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), 12)
	if err != nil {
		return nil, err
	}

	domain.Matches(string(hash), dto.Password)

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeActivation, user.Id)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
	}

	token, err := a.NewToken(ctx, user.Id, 3*24*time.Hour, domain.ScopeActivation)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
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

	return user, nil
}

func (a *AuthService) CreateResetPasswordToken(ctx context.Context, dto domain.UserDTO) (*domain.User, error) {
	user, err := a.identiRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return nil, fmt.Errorf("user not exit")
	}

	err = a.tokenRepo.DeleteAllForUser(ctx, domain.ScopeResetPassword, user.Id)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
	}

	token, err := a.NewToken(ctx, user.Id, 3*24*time.Hour, domain.ScopeResetPassword)
	if err != nil {
		return nil, fmt.Errorf("internal server error")
	}

	go func(user domain.User, resetPasswordToken string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &notificationv1.ResetPasswordMessageRequest{
			IdentityUuid:       user.Id.String(),
			Email:              user.Email,
			ResetPasswordToken: resetPasswordToken,
		}

		if _, err := a.notificationClient.SendResetPasswordMessage(ctx, req); err != nil {
			a.logger.Error(
				"failed to send reset password message",
				"error", err,
				"user_id", user.Id,
			)
		}
	}(*user, token.Plaintext)

	return user, nil
}

func (a *AuthService) LogOut(ctx context.Context, dto domain.UserLogOutDTO) error {
	err := a.tokenRepo.DeleteTokenForUser(ctx, dto.Refresh, dto.Id)
	if err != nil {
		return fmt.Errorf("internal server error")
	}

	return nil
}
