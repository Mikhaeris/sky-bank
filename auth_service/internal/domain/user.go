package domain

import (
	"errors"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id           uuid.UUID
	Email        string
	PasswordHash string
	Activated    bool
	Version      int
}

type UserDTO struct {
	Email    string
	Password string
}

type ActivateUserDTO struct {
	TokenPlaintext string
}

type ResetPasswordDTO struct {
	TokenPlaintext string
	Password       string
}

type UserLogOutDTO struct {
	Id uuid.UUID
	TokensDTO
}

func NewUser(dto UserDTO, userUUID uuid.UUID, passwordHash string) *User {
	return &User{
		Id:           userUUID,
		Email:        dto.Email,
		PasswordHash: passwordHash,
	}
}

func Matches(passwordHash, plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}
