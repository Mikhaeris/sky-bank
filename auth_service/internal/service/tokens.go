package service

import (
	"context"
	"fmt"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
	"github.com/mikhaeris/sky-bank/auth_service/internal/utils"
)

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

	token, err := utils.CreateToken(user)
	if err != nil {
		return "", err
	}

	return token, nil
}
