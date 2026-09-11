package service

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

func (a *AuthService) NewToken(ctx context.Context, identityId uuid.UUID, ttl time.Duration, scope string) (*domain.Token, error) {
	token := domain.GenerateToken(identityId, ttl, scope)

	err := a.tokenRepo.Insert(ctx, token)
	return token, err
}

func (a *AuthService) CreateToken(ctx context.Context, dto domain.UserDTO) (string, error) {

	// validate

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return "", err
	}

	// if user.Activated == false {
	// 	return "", fmt.Errorf("user not activated")
	// }

	// check passwords
	match, err := domain.Matches(user.PasswordHash, dto.Password)
	if err != nil {
		return "", err
	}

	if !match {
		return "", fmt.Errorf("user data is incorrect")
	}

	token, err := a.jwtKey.CreateToken(user)
	if err != nil {
		return "", err
	}

	return token, nil
}
