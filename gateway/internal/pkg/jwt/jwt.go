package jwt

import (
	"crypto"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Email string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

type TokenVerifier struct {
	publicKey crypto.PublicKey
}

func NewTokenVerifier(publicKeyPath string) (*TokenVerifier, error) {
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	key, err := jwt.ParseEdPublicKeyFromPEM(data)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &TokenVerifier{
		publicKey: key,
	}, nil
}

func (v *TokenVerifier) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(*jwt.Token) (any, error) {
			return v.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Subject == "" {
		return nil, errors.New("missing subject")
	}

	return claims, nil
}
