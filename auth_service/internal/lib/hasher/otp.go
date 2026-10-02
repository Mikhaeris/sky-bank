package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"os"
)

const codeHashSecretSize = 32

type CodeHasher struct {
	secret []byte
}

func NewCodeHasher(pathToSecret string) (*CodeHasher, error) {
	secret, err := os.ReadFile(pathToSecret)
	if err != nil {
		return nil, fmt.Errorf("read code hashing secret: %w", err)
	}

	if len(secret) != codeHashSecretSize {
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
