package domain

import (
	"time"
	"uuid"
)

const BirthDateLayout = "2006/01/02"

const (
	KYCStatusNotStarted string = "not_started"
	KYCStatusPending    string = "pending"
	KYCStatusVerified   string = "verified"
	KYCStatusRejected   string = "rejected"
)

type Customer struct {
	ID            uuid.UUID
	Email         string
	EmailVerified bool
	FirstName     string
	LastName      string
	MiddleName    string
	BirthDate     time.Time
	Gender        string
	KycStatus     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StartEmailVerificationDTO struct {
	Email string
}

type CompleteEmailVerificationDTO struct {
	ChallengeID string
	OtpCode     string
}
