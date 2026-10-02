package domain

import (
	"time"
	"uuid"
)

type OTPOutbox struct {
	EventID        uuid.UUID
	ChallengeID    uuid.UUID
	EncryptedEvent []byte
	TraceParent    string
	TraceState     string
	ExpiresAt      time.Time
	LeasedUntil    time.Time
}
