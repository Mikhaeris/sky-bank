package utils

import (
	"crypto"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mikhaeris/bank-test/auth_service/internal/models"
)

const privKeyPath = "../keys/private.pem"

var signKey crypto.PrivateKey

func init() {
	signBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		log.Fatal(err)
	}

	signKey, err = jwt.ParseEdPrivateKeyFromPEM(signBytes)
	if err != nil {
		log.Fatal(err)
	}
}

type UserInfo struct {
	Email string
}

type UserClaims struct {
	jwt.RegisteredClaims
	UserInfo
}

func CreateToken(user *models.User) (string, error) {
	t := jwt.New(jwt.SigningMethodEdDSA)

	t.Claims = &UserClaims{
		jwt.RegisteredClaims{
			Subject: user.ID.String(),

			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
		UserInfo{
			Email: user.Email,
		},
	}

	return t.SignedString(signKey)
}

const pubKeyPath = "../keys/public.pem"

var pubKey crypto.PublicKey

func init() {
	pubBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		log.Fatal(err)
	}

	pubKey, err = jwt.ParseEdPublicKeyFromPEM(pubBytes)
	if err != nil {
		log.Fatal(err)
	}
}

func GetToken(tokenString string) (*UserClaims, error) {
	claims := &UserClaims{}

	_, err := jwt.ParseWithClaims(
		tokenString,
		claims,

		func(token *jwt.Token) (any, error) {
			return pubKey, nil
		},
		jwt.WithValidMethods([]string{
			jwt.SigningMethodEdDSA.Alg(),
		}),
	)

	if err != nil {
		return nil, err
	}

	return claims, nil
}
