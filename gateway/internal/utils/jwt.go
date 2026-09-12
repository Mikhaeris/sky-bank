package utils

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

const (
	authorizationMetadata = "authorization"
	UserIDMetadata        = "x-user-id"
)

var (
	ErrAuthorizationMissing = errors.New("authorization metadata missing")
	ErrInvalidAuthScheme    = errors.New("invalid authorization scheme")
	ErrTokenMissing         = errors.New("access token missing")
	ErrSubjectMissing       = errors.New("token subject missing")
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

	publicKey, err := jwt.ParseEdPublicKeyFromPEM(data)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &TokenVerifier{
		publicKey: publicKey,
	}, nil
}

func (v *TokenVerifier) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			return v.publicKey, nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodEdDSA.Alg(),
		}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Subject == "" {
		return nil, ErrSubjectMissing
	}

	return claims, nil
}

func ExtractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return "", ErrAuthorizationMissing
	}

	values := md.Get(authorizationMetadata)
	if len(values) == 0 {
		return "", ErrAuthorizationMissing
	}

	for _, value := range values {
		parts := strings.Fields(value)

		if len(parts) != 2 {
			continue
		}

		if !strings.EqualFold(parts[0], "Bearer") {
			continue
		}

		if parts[1] == "" {
			return "", ErrTokenMissing
		}

		return parts[1], nil
	}

	return "", ErrInvalidAuthScheme
}
