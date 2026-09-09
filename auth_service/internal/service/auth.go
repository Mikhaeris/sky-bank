package service

import (
	"context"
	"uuid"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
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

	a.logger.Info("ok", user)

	// send user_service information

	// generate token for email

	// send confirmation email

	return user.ID, nil
}

func (a *AuthService) ActivateUser(ctx context.Context, dto domain.ActivateUserDTO) (domain.User, error) {
	return domain.User{}, nil
}
