package service

import (
	"context"
	"time"
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

	// send user_service information to create

	// generate token for email
	token, err := a.NewToken(ctx, userUUID, 3*24*time.Hour, domain.ScopeActivation)

	// send confirmation email
	go func(token []byte) {

	}(token.Hash)

	return user.ID, nil
}

func (a *AuthService) ActivateUser(ctx context.Context, dto domain.ActivateUserDTO) (domain.User, error) {
	return domain.User{}, nil
}
