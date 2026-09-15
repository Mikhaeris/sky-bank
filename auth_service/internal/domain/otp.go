package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"time"
)

type CodePurpose string

const (
	CodePurposeAuthentication CodePurpose = "authentication"
)

type CodeTTL = time.Duration

const (
	CodeTTLAuthentication CodeTTL = 5 * time.Minute
)

type OneTimeCode struct {
	Email         string
	Purpose       CodePurpose
	CodePlaintext string
	CodeHash      []byte
	ExpiresAt     time.Time
}

func GenerateCode(hasher *CodeHasher, email string, purpose CodePurpose, ttl time.Duration) (*OneTimeCode, error) {
	const max = 1_000_000

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}

	plaintext := fmt.Sprintf("%06d", n.Int64())

	return &OneTimeCode{
		Email:         email,
		Purpose:       purpose,
		CodePlaintext: plaintext,
		CodeHash:      hasher.Hash(plaintext),
		ExpiresAt:     time.Now().Add(ttl),
	}, nil
}

type OtpDto struct {
	Email         string
	CodePlaintext string
}

type CodeHasher struct {
	secret []byte
}

func NewCodeHasher(pathToSecret string) (*CodeHasher, error) {
	secret, err := os.ReadFile(pathToSecret)
	if err != nil {
		return nil, fmt.Errorf("read code hashing secret: %w", err)
	}

	if len(secret) != 32 {
		return nil, fmt.Errorf(
			"invalid code hashing secret length: %d",
			len(secret),
		)
	}

	return &CodeHasher{
		secret: secret,
	}, nil
}

func (c *CodeHasher) Hash(code string) []byte {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(code))
	return mac.Sum(nil)
}

func (c *CodeHasher) Verify(code string, expectedHash []byte) bool {
	return hmac.Equal(c.Hash(code), expectedHash)
}
