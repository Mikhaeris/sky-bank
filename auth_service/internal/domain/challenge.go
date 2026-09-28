package domain

import (
	"time"
	"uuid"
)

type CodeTTL = time.Duration

const (
	CodeTTLAuthentication CodeTTL = 5 * time.Minute
)

type OtpChannel string

const (
	OtpChannelEmail OtpChannel = "email"
	OtpChannelSMS   OtpChannel = "sms"
)

type CodePurpose string

const (
	CodePurposeAuthentication    CodePurpose = "authentication"
	CodePurposeEmailVerification CodePurpose = "email verification"
)

type Challenge struct {
	ID             uuid.UUID
	Destination    string
	Channel        OtpChannel
	Purpose        CodePurpose
	CodeHash       []byte
	ExpiresAt      time.Time
	FailedAttempts int
}

type OtpDTO struct {
	ChallengeID   uuid.UUID
	CodePlaintext string
}

type IssueChallengeDTO struct {
	Destination string
	Channel     OtpChannel
	Purpose     CodePurpose
}

type ConsumeChallengeDTO struct {
	ChallengeID         uuid.UUID
	Code                string
	ExpectedPurpose     CodePurpose
	ExpectedDestination *string
}

type VerifiedChallenge struct {
	Destination string
	Channel     OtpChannel
	Purpose     CodePurpose
}
