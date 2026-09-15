package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"time"

	"uuid"
)

const RefreshTokenTTL = 30 * 24 * time.Hour

type Session struct {
	ID                    uuid.UUID
	IdentityID            uuid.UUID
	RefreshTokenPlaintext string
	RefreshTokenHash      []byte
	CreatedAt             time.Time
	ExpiresAt             time.Time
	LastUsedAt            time.Time
}

func NewSession(identityID uuid.UUID, ttl time.Duration) *Session {
	session := &Session{
		ID:         uuid.New(),
		IdentityID: identityID,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		LastUsedAt: time.Now(),
	}

	session.RotateRefreshToken()

	return session
}

func (s *Session) RotateRefreshToken() {
	token := rand.Text()

	s.RefreshTokenPlaintext = token
	s.RefreshTokenHash = HashRefreshToken(token)
	s.LastUsedAt = time.Now()
}

func HashRefreshToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

type RevokeSessionDTO struct {
	SessionID uuid.UUID
}
