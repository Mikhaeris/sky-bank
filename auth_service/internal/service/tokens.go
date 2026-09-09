package service

import (
	"context"
	"fmt"

	"github.com/mikhaeris/bank-test/auth_service/internal/models"
	"github.com/mikhaeris/bank-test/auth_service/internal/utils"
)

func (a *AuthService) CreateToken(ctx context.Context, dto models.UserDTO) (string, error) {

	// validate

	user, err := a.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		return "", err
	}

	// if user.Activated == false {
	// 	return "", fmt.Errorf("user not activated")
	// }

	// check passwords
	match, err := models.Matches(user.PasswordHash, dto.Password)
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
