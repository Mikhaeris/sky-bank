package domain

import "time"

type OTPLimitState struct {
	Destination  string
	Channel      OtpChannel
	Purpose      CodePurpose
	LastIssuedAt *time.Time
	IssueCount   int
	UpdatedAt    time.Time
}
