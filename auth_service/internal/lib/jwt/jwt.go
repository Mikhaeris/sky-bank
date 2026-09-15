package jwt

import (
	"crypto"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type Keys struct {
	signKey crypto.PrivateKey
	ttl     time.Duration
}

func NewKeys(privKeyPath string, ttl time.Duration) (*Keys, error) {
	signBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	signKey, err := jwt.ParseEdPrivateKeyFromPEM(signBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &Keys{
		signKey: signKey,
		ttl:     ttl,
	}, nil
}

type UserInfo struct {
	Email string
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserInfo
}

func (k *Keys) CreateToken(identity *domain.Identity) (string, error) {
	t := jwt.New(jwt.SigningMethodEdDSA)

	t.Claims = &UserClaims{
		jwt.RegisteredClaims{
			Subject: identity.ID.String(),

			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(k.ttl)),
		},
		UserInfo{
			Email: identity.Email,
		},
	}

	return t.SignedString(k.signKey)
}
