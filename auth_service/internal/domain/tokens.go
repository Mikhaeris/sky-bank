package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"time"
	"uuid"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext  string
	Hash       []byte
	IdentityId uuid.UUID
	Expiry     time.Time
	Scope      string
}

func GenerateToken(identityId uuid.UUID, ttl time.Duration, scope string) *Token {
	s := rand.Text()
	hash := sha256.Sum256([]byte(s))

	return &Token{
		Plaintext:  s,
		Hash:       hash[:],
		IdentityId: identityId,
		Expiry:     time.Now().Add(ttl),
		Scope:      scope,
	}
}
