package domain

import "time"

type OTPLimitState struct {
	Destination            string
	Channel                OtpChannel
	Purpose                CodePurpose
	LastIssuedAt           *time.Time
	IssueWindowStartedAt   *time.Time
	IssueCount             int
	FailureWindowStartedAt *time.Time
	FailureCount           int
	BlockedUntil           *time.Time
	UpdatedAt              time.Time
}
