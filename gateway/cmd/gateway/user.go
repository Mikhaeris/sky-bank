package main

import (
	"errors"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Activated    bool
}

type UserDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ActivateUserDTO struct {
	activateToken string
}

func NewUser(dto UserDTO, userUUID uuid.UUID, passwordHash string) *User {
	return &User{
		ID:           userUUID,
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
