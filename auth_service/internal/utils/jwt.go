package utils

import (
	"crypto"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mikhaeris/sky-bank/auth_service/internal/domain"
)

type Keys struct {
	signKey crypto.PrivateKey
	ttl     time.Duration
}

func NewKeys(privKeyPath string, ttl time.Duration, logger *slog.Logger) *Keys {
	signBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		logger.Error("can't get prevKey file", "error", err)
		os.Exit(1)
	}

	signKey, err := jwt.ParseEdPrivateKeyFromPEM(signBytes)
	if err != nil {
		logger.Error("error while parsing private key", "error", err)
		os.Exit(1)
	}

	return &Keys{
		signKey: signKey,
		ttl:     ttl,
	}
}

type UserInfo struct {
	Email string
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserInfo
}

func (k *Keys) CreateToken(user *domain.User) (string, error) {
	t := jwt.New(jwt.SigningMethodEdDSA)

	t.Claims = &UserClaims{
		jwt.RegisteredClaims{
			Subject: user.ID.String(),

			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(k.ttl)),
		},
		UserInfo{
			Email: user.Email,
		},
	}

	return t.SignedString(k.signKey)
}

// const pubKeyPath = "../keys/public.pem"

// var pubKey crypto.PublicKey

// func init() {
// 	pubBytes, err := os.ReadFile(pubKeyPath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	pubKey, err = jwt.ParseEdPublicKeyFromPEM(pubBytes)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

// func GetToken(tokenString string) (*UserClaims, error) {
// 	claims := &UserClaims{}

// 	_, err := jwt.ParseWithClaims(
// 		tokenString,
// 		claims,

// 		func(token *jwt.Token) (any, error) {
// 			return pubKey, nil
// 		},
// 		jwt.WithValidMethods([]string{
// 			jwt.SigningMethodEdDSA.Alg(),
// 		}),
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return claims, nil
// }
